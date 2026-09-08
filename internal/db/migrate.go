package db

import (
	"database/sql"
	"fmt"

	"github.com/deliium/drawing-board/internal/db/migrations"
)

func ensureSchemaMigrations(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`)
	return err
}

func currentSchemaVersion(db *sql.DB) (int, error) {
	var v sql.NullInt64
	err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

func hasBoardTables(db *sql.DB) (bool, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(1) FROM sqlite_master
		WHERE type='table' AND name IN ('users','strokes','stroke_points')
	`).Scan(&n)
	if err != nil {
		return false, err
	}
	return n >= 3, nil
}

// runMigrations applies pending numbered migrations fail-closed. One transaction per version.
func runMigrations(db *sql.DB) error {
	if err := ensureSchemaMigrations(db); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	cur, err := currentSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if cur == 0 {
		legacy, err := hasBoardTables(db)
		if err != nil {
			return err
		}
		if legacy {
			dbLog("WARN", "[db.migrate] legacy board schema detected; stamping via idempotent baseline")
		}
	}

	pending := 0
	from := cur
	for _, m := range migrations.All() {
		if m.Version <= cur {
			continue
		}
		pending++
		dbLog("DEBUG", "[db.migrate] apply version=%d name=%s", m.Version, m.Name)
		if err := applyMigration(db, m); err != nil {
			dbLog("ERROR", "[db.migrate] failed version=%d name=%s err=%v", m.Version, m.Name, err)
			return err
		}
		if m.Version == 1 {
			dbLog("INFO", "[db.migrate] baseline ready version=1")
		}
		cur = m.Version
	}

	if pending == 0 {
		dbLog("INFO", "[db.migrate] up-to-date version=%d", cur)
	} else {
		dbLog("INFO", "[db.migrate] applied %d→%d", from, cur)
	}
	return nil
}

func applyMigration(db *sql.DB, m migrations.Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := m.Up(tx); err != nil {
		return fmt.Errorf("up version=%d: %w", m.Version, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations(version, name) VALUES(?, ?)`,
		m.Version, m.Name,
	); err != nil {
		return fmt.Errorf("record version=%d: %w", m.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// SchemaVersion returns the highest applied migration version (0 if none).
func (s *Store) SchemaVersion() (int, error) {
	return currentSchemaVersion(s.SQL)
}
