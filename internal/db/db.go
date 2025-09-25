package db

import (
	"database/sql"
	"errors"
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
	ID int64
	UserID int64
	Color string
	Width int
	StartedAtUnixMs int64
	Points []StrokePoint
	CreatedAt time.Time
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil { return nil, err }
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil { return nil, err }
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil { return nil, err }
	if err := migrate(db); err != nil { return nil, err }
	return &Store{SQL: db}, nil
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
	return err
}

func (s *Store) CreateUser(email, passwordHash string) (int64, error) {
	res, err := s.SQL.Exec("INSERT INTO users(email, password_hash) VALUES(?, ?)", email, passwordHash)
	if err != nil { return 0, err }
	return res.LastInsertId()
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
	tx, err := s.SQL.Begin()
	if err != nil { return 0, err }
	defer func(){ if err != nil { _ = tx.Rollback() } }()
	res, err := tx.Exec("INSERT INTO strokes(user_id, color, width, started_at_unix_ms) VALUES(?, ?, ?, ?)", userID, color, width, startedAtUnixMs)
	if err != nil { return 0, err }
	strokeID, err := res.LastInsertId()
	if err != nil { return 0, err }
	if len(points) > 0 {
		stmt, err := tx.Prepare("INSERT INTO stroke_points(stroke_id, x, y) VALUES(?, ?, ?)")
		if err != nil { return 0, err }
		for _, p := range points {
			if _, err := stmt.Exec(strokeID, p.X, p.Y); err != nil { _ = stmt.Close(); return 0, err }
		}
		_ = stmt.Close()
	}
	if err := tx.Commit(); err != nil { return 0, err }
	return strokeID, nil
}

func (s *Store) ListStrokesByUser(userID int64) ([]Stroke, error) {
	rows, err := s.SQL.Query("SELECT id, color, width, started_at_unix_ms, created_at FROM strokes WHERE user_id = ? ORDER BY id", userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Stroke
	for rows.Next() {
		var st Stroke
		st.UserID = userID
		if err := rows.Scan(&st.ID, &st.Color, &st.Width, &st.StartedAtUnixMs, &st.CreatedAt); err != nil { return nil, err }
		pr, err := s.SQL.Query("SELECT x, y FROM stroke_points WHERE stroke_id = ? ORDER BY id", st.ID)
		if err != nil { return nil, err }
		for pr.Next() {
			var x, y float64
			if err := pr.Scan(&x, &y); err != nil { pr.Close(); return nil, err }
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
