package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/deliium/drawing-board/internal/ids"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	SQL *sql.DB
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type StrokePoint struct {
	X float64
	Y float64
}

type Stroke struct {
	ID              string
	UserID          string
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
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if enableFK {
		if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
			return db, fmt.Errorf("pragma foreign_keys: %w", err)
		}
	}
	// WAL can fail on some filesystems / environments (e.g. limited locking). Prefer it, but don't hard-fail.
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		// Try to explicitly switch back to DELETE; if that also fails, continue with SQLite defaults.
		_, _ = db.Exec("PRAGMA journal_mode=DELETE;")
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return db, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if err := runMigrations(db); err != nil {
		return db, fmt.Errorf("migrate: %w", err)
	}
	store := &Store{SQL: db}
	if err := SeedHiragana5(store); err != nil {
		return db, fmt.Errorf("seed: %w", err)
	}
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

func (s *Store) CreateUser(email, passwordHash string) (string, error) {
	id := ids.New()
	_, err := s.SQL.Exec("INSERT INTO users(id, email, password_hash) VALUES(?, ?, ?)", id, email, passwordHash)
	if err != nil {
		return "", err
	}
	return id, nil
}

// DeleteUser removes a user by id. Used to roll back a failed registration after insert.
func (s *Store) DeleteUser(userID string) error {
	res, err := s.SQL.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("DeleteUser: no user id=%s", userID)
	}
	return nil
}

// UpdateUserPasswordHash rewrites users.password_hash (e.g. future password-change flows).
func (s *Store) UpdateUserPasswordHash(userID string, passwordHash string) error {
	res, err := s.SQL.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("UpdateUserPasswordHash: no user id=%s", userID)
	}
	return nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	row := s.SQL.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE email = ?", email)
	u := User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(id string) (*User, error) {
	row := s.SQL.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE id = ?", id)
	u := User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *Store) SaveStroke(userID string, color string, width int, startedAtUnixMs int64, points []StrokePoint) (string, error) {
	id, _, err := s.SaveStrokeIdempotent(userID, "", color, width, startedAtUnixMs, points)
	return id, err
}

// SaveStrokeIdempotent inserts a stroke. When opID is non-empty, a second insert with the same
// (user_id, op_id) returns the existing stroke id with created=false.
func (s *Store) SaveStrokeIdempotent(userID string, opID, color string, width int, startedAtUnixMs int64, points []StrokePoint) (strokeID string, created bool, err error) {
	tx, err := s.SQL.Begin()
	if err != nil {
		return "", false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	strokeID = ids.New()
	if opID == "" {
		_, err = tx.Exec(
			"INSERT INTO strokes(id, user_id, color, width, started_at_unix_ms) VALUES(?, ?, ?, ?, ?)",
			strokeID, userID, color, width, startedAtUnixMs,
		)
	} else {
		_, err = tx.Exec(
			"INSERT INTO strokes(id, user_id, color, width, started_at_unix_ms, op_id) VALUES(?, ?, ?, ?, ?, ?)",
			strokeID, userID, color, width, startedAtUnixMs, opID,
		)
	}
	if err != nil {
		if opID != "" && isUniqueConstraintErr(err) {
			_ = tx.Rollback()
			existing, lookupErr := s.strokeIDByOp(userID, opID)
			if lookupErr != nil {
				return "", false, lookupErr
			}
			return existing, false, nil
		}
		return "", false, err
	}
	if len(points) > 0 {
		stmt, prepErr := tx.Prepare("INSERT INTO stroke_points(id, stroke_id, x, y) VALUES(?, ?, ?, ?)")
		if prepErr != nil {
			err = prepErr
			return "", false, err
		}
		for _, p := range points {
			if _, err = stmt.Exec(ids.New(), strokeID, p.X, p.Y); err != nil {
				_ = stmt.Close()
				return "", false, err
			}
		}
		_ = stmt.Close()
	}
	if err = tx.Commit(); err != nil {
		return "", false, err
	}
	return strokeID, true, nil
}

func (s *Store) strokeIDByOp(userID string, opID string) (string, error) {
	var id string
	err := s.SQL.QueryRow(
		"SELECT id FROM strokes WHERE user_id = ? AND op_id = ?",
		userID, opID,
	).Scan(&id)
	if err != nil {
		return "", err
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

func (s *Store) ListStrokesByUser(userID string) ([]Stroke, error) {
	rows, err := s.SQL.Query("SELECT id, color, width, started_at_unix_ms, created_at, COALESCE(op_id, '') FROM strokes WHERE user_id = ? ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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
				_ = pr.Close()
				return nil, err
			}
			st.Points = append(st.Points, StrokePoint{X: x, Y: y})
		}
		_ = pr.Close()
		out = append(out, st)
	}
	return out, nil
}

func (s *Store) ClearStrokesByUser(userID string) error {
	_, err := s.SQL.Exec("DELETE FROM strokes WHERE user_id = ?", userID)
	return err
}

func (s *Store) DeleteStroke(userID string, strokeID string) error {
	_, err := s.SQL.Exec("DELETE FROM strokes WHERE id = ? AND user_id = ?", strokeID, userID)
	return err
}
