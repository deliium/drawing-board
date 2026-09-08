package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	SQL *sql.DB
}

type User struct {
	ID int64
	Email string
	PasswordHash string
	CreatedAt time.Time
}

type StrokePoint struct { X float64; Y float64 }

type Stroke struct {
	ID              int64
	UserID          int64
	Color           string
	Width           int
	StartedAtUnixMs int64
	OpID            string
	Points          []StrokePoint
	CreatedAt       time.Time
}

func Open(path string) (*Store, error) {
	filename, enableFK := normalizeSQLitePath(path)
	db, err := openAndInit(filename, enableFK)
	if err == nil {
		return &Store{SQL: db}, nil
	}

	// In some environments stale -wal/-shm sidecar files can break opening the DB (disk I/O error).
	// If that happens, try removing the sidecars and retry once.
	if isSQLiteSidecarError(err) {
		_ = db.Close()
		_ = os.Remove(filename + "-wal")
		_ = os.Remove(filename + "-shm")
		db2, err2 := openAndInit(filename, enableFK)
		if err2 == nil {
			return &Store{SQL: db2}, nil
		}
		return nil, err2
	}

	_ = db.Close()
	return nil, err
}

func openAndInit(filename string, enableFK bool) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil { return nil, err }

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if enableFK {
		if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil { return db, fmt.Errorf("pragma foreign_keys: %w", err) }
	}
	// WAL can fail on some filesystems / environments (e.g. limited locking). Prefer it, but don't hard-fail.
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		// Try to explicitly switch back to DELETE; if that also fails, continue with SQLite defaults.
		_, _ = db.Exec("PRAGMA journal_mode=DELETE;")
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil { return db, fmt.Errorf("pragma busy_timeout: %w", err) }
	if err := migrate(db); err != nil { return db, fmt.Errorf("migrate: %w", err) }
	return db, nil
}

func isSQLiteSidecarError(err error) bool {
	// Keep it conservative: only retry on the exact class of errors we saw.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "disk i/o error") && strings.Contains(s, "no such file or directory")
}

func normalizeSQLitePath(dsnOrPath string) (filename string, enableFK bool) {
	s := strings.TrimSpace(dsnOrPath)
	if s == "" {
		return "data.db", true
	}

	// Handle legacy/default form like: file:data.db?_fk=1
	// We normalize it to a plain filename so it works consistently across environments.
	if strings.HasPrefix(s, "file:") && strings.Contains(s, "?") {
		if u, err := url.Parse(s); err == nil && u.Scheme == "file" {
			if v := u.Query().Get("_fk"); v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "on") {
				enableFK = true
			}

			// For "file:data.db?...": url.Parse uses Opaque for the path part.
			if u.Opaque != "" {
				return u.Opaque, enableFK
			}
			if u.Path != "" {
				return u.Path, enableFK
			}
		}
	}

	return s, true
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS strokes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		color TEXT NOT NULL,
		width INTEGER NOT NULL,
		started_at_unix_ms INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS stroke_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		stroke_id INTEGER NOT NULL REFERENCES strokes(id) ON DELETE CASCADE,
		x REAL NOT NULL,
		y REAL NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_strokes_user ON strokes(user_id);
	CREATE INDEX IF NOT EXISTS idx_stroke_points_stroke ON stroke_points(stroke_id);
	`)
	if err != nil {
		return err
	}
	// Existing DBs: add nullable op_id for idempotent WS creates (ignore if already present).
	if _, err := db.Exec(`ALTER TABLE strokes ADD COLUMN op_id TEXT`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_strokes_user_op ON strokes(user_id, op_id) WHERE op_id IS NOT NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS user_board_state (
		user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		board_rev INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS stroke_op_tombstones (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		op_id TEXT NOT NULL,
		reason TEXT NOT NULL,
		at_rev INTEGER NOT NULL,
		PRIMARY KEY (user_id, op_id)
	);
	CREATE TABLE IF NOT EXISTS board_ops (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		op_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		at_rev INTEGER NOT NULL,
		PRIMARY KEY (user_id, op_id)
	);
	`)
	return err
}

func (s *Store) CreateUser(email, passwordHash string) (int64, error) {
	res, err := s.SQL.Exec("INSERT INTO users(email, password_hash) VALUES(?, ?)", email, passwordHash)
	if err != nil { return 0, err }
	return res.LastInsertId()
}

// UpdateUserPasswordHash rewrites users.password_hash for transparent legacy→bcrypt upgrades.
func (s *Store) UpdateUserPasswordHash(userID int64, passwordHash string) error {
	res, err := s.SQL.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("UpdateUserPasswordHash: no user id=%d", userID)
	}
	return nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	row := s.SQL.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE email = ?", email)
	u := User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return nil, nil }
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	row := s.SQL.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE id = ?", id)
	u := User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return nil, nil }
		return nil, err
	}
	return &u, nil
}

func (s *Store) SaveStroke(userID int64, color string, width int, startedAtUnixMs int64, points []StrokePoint) (int64, error) {
	id, _, err := s.SaveStrokeIdempotent(userID, "", color, width, startedAtUnixMs, points)
	return id, err
}

// SaveStrokeIdempotent inserts a stroke. When opID is non-empty, a second insert with the same
// (user_id, op_id) returns the existing stroke id with created=false.
func (s *Store) SaveStrokeIdempotent(userID int64, opID, color string, width int, startedAtUnixMs int64, points []StrokePoint) (strokeID int64, created bool, err error) {
	tx, err := s.SQL.Begin()
	if err != nil {
		return 0, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var res sql.Result
	if opID == "" {
		res, err = tx.Exec(
			"INSERT INTO strokes(user_id, color, width, started_at_unix_ms) VALUES(?, ?, ?, ?)",
			userID, color, width, startedAtUnixMs,
		)
	} else {
		res, err = tx.Exec(
			"INSERT INTO strokes(user_id, color, width, started_at_unix_ms, op_id) VALUES(?, ?, ?, ?, ?)",
			userID, color, width, startedAtUnixMs, opID,
		)
	}
	if err != nil {
		if opID != "" && isUniqueConstraintErr(err) {
			_ = tx.Rollback()
			existing, lookupErr := s.strokeIDByOp(userID, opID)
			if lookupErr != nil {
				return 0, false, lookupErr
			}
			return existing, false, nil
		}
		return 0, false, err
	}
	strokeID, err = res.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	if len(points) > 0 {
		stmt, prepErr := tx.Prepare("INSERT INTO stroke_points(stroke_id, x, y) VALUES(?, ?, ?)")
		if prepErr != nil {
			err = prepErr
			return 0, false, err
		}
		for _, p := range points {
			if _, err = stmt.Exec(strokeID, p.X, p.Y); err != nil {
				_ = stmt.Close()
				return 0, false, err
			}
		}
		_ = stmt.Close()
	}
	if err = tx.Commit(); err != nil {
		return 0, false, err
	}
	return strokeID, true, nil
}

func (s *Store) strokeIDByOp(userID int64, opID string) (int64, error) {
	var id int64
	err := s.SQL.QueryRow(
		"SELECT id FROM strokes WHERE user_id = ? AND op_id = ?",
		userID, opID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}

func (s *Store) ListStrokesByUser(userID int64) ([]Stroke, error) {
	rows, err := s.SQL.Query("SELECT id, color, width, started_at_unix_ms, created_at, COALESCE(op_id, '') FROM strokes WHERE user_id = ? ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Stroke
	for rows.Next() {
		var st Stroke
		st.UserID = userID
		if err := rows.Scan(&st.ID, &st.Color, &st.Width, &st.StartedAtUnixMs, &st.CreatedAt, &st.OpID); err != nil {
			return nil, err
		}
		pr, err := s.SQL.Query("SELECT x, y FROM stroke_points WHERE stroke_id = ? ORDER BY id", st.ID)
		if err != nil {
			return nil, err
		}
		for pr.Next() {
			var x, y float64
			if err := pr.Scan(&x, &y); err != nil {
				pr.Close()
				return nil, err
			}
			st.Points = append(st.Points, StrokePoint{X: x, Y: y})
		}
		pr.Close()
		out = append(out, st)
	}
	return out, nil
}

func (s *Store) ClearStrokesByUser(userID int64) error {
	_, err := s.SQL.Exec("DELETE FROM strokes WHERE user_id = ?", userID)
	return err
}

func (s *Store) DeleteStroke(userID int64, strokeID int64) error {
	_, err := s.SQL.Exec("DELETE FROM strokes WHERE id = ? AND user_id = ?", strokeID, userID)
	return err
}
