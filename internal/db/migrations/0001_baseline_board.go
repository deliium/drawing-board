package migrations

import (
	"database/sql"
	"strings"
)

// up0001BaselineBoard encodes the free-board schema that existed before versioned migrations.
// Idempotent CREATE IF NOT EXISTS / safe ALTER so legacy DBs stamp without data loss.
func up0001BaselineBoard(tx *sql.Tx) error {
	_, err := tx.Exec(`
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
	if _, err := tx.Exec(`ALTER TABLE strokes ADD COLUMN op_id TEXT`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_strokes_user_op ON strokes(user_id, op_id) WHERE op_id IS NOT NULL`); err != nil {
		return err
	}
	_, err = tx.Exec(`
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
