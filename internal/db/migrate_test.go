package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/deliium/drawing-board/internal/db/migrations"
	"github.com/deliium/drawing-board/internal/recognize"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateFreshOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.SQL.Close()

	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if ver != migrations.LatestVersion() {
		t.Fatalf("version=%d want %d", ver, migrations.LatestVersion())
	}

	for _, name := range []string{
		"characters", "lessons", "lesson_characters", "practice_attempts",
		"attempt_strokes", "attempt_stroke_points", "assessment_results",
		"assessment_feedback", "user_character_progress", "schema_migrations",
	} {
		var n int
		if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n); err != nil || n != 1 {
			t.Fatalf("missing table %s: n=%d err=%v", name, n, err)
		}
	}

	var charCount int
	if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM characters WHERE set_id=?`, recognize.SetIDHiragana5).Scan(&charCount); err != nil {
		t.Fatalf("count characters: %v", err)
	}
	if charCount != 5 {
		t.Fatalf("seed characters=%d want 5", charCount)
	}

	// Re-open is idempotent.
	store.SQL.Close()
	store2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer store2.SQL.Close()
	ver2, _ := store2.SchemaVersion()
	if ver2 != ver {
		t.Fatalf("re-open version=%d want %d", ver2, ver)
	}
	if err := store2.SQL.QueryRow(`SELECT COUNT(1) FROM characters WHERE set_id=?`, recognize.SetIDHiragana5).Scan(&charCount); err != nil || charCount != 5 {
		t.Fatalf("seed after re-open count=%d err=%v", charCount, err)
	}
}

func TestMigrateLegacyBootstrapPreservesStrokes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("fk: %v", err)
	}
	// Pre-versioned board schema (no schema_migrations).
	_, err = raw.Exec(`
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE strokes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		color TEXT NOT NULL,
		width INTEGER NOT NULL,
		started_at_unix_ms INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		op_id TEXT
	);
	CREATE TABLE stroke_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		stroke_id INTEGER NOT NULL REFERENCES strokes(id) ON DELETE CASCADE,
		x REAL NOT NULL,
		y REAL NOT NULL
	);
	CREATE TABLE user_board_state (
		user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		board_rev INTEGER NOT NULL DEFAULT 0
	);
	`)
	if err != nil {
		t.Fatalf("legacy ddl: %v", err)
	}
	res, err := raw.Exec(`INSERT INTO users(email, password_hash) VALUES('legacy@example.com','hash')`)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := res.LastInsertId()
	sres, err := raw.Exec(`INSERT INTO strokes(user_id, color, width, started_at_unix_ms, op_id) VALUES(?,?,?,?,?)`, uid, "#000000", 2, 1, "op-keep")
	if err != nil {
		t.Fatalf("stroke: %v", err)
	}
	sid, _ := sres.LastInsertId()
	if _, err := raw.Exec(`INSERT INTO stroke_points(stroke_id, x, y) VALUES(?,?,?)`, sid, 10.5, 20.5); err != nil {
		t.Fatalf("point: %v", err)
	}
	_ = raw.Close()

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open legacy: %v", err)
	}
	defer store.SQL.Close()

	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if ver != migrations.LatestVersion() {
		t.Fatalf("version=%d want %d", ver, migrations.LatestVersion())
	}

	strokes, err := store.ListStrokesByUser(uid)
	if err != nil {
		t.Fatalf("ListStrokes: %v", err)
	}
	if len(strokes) != 1 {
		t.Fatalf("strokes survived=%d want 1", len(strokes))
	}
	if strokes[0].OpID != "op-keep" || len(strokes[0].Points) != 1 {
		t.Fatalf("stroke payload corrupted: %+v", strokes[0])
	}
	if strokes[0].Points[0].X != 10.5 || strokes[0].Points[0].Y != 20.5 {
		t.Fatalf("points: %+v", strokes[0].Points)
	}

	var charCount int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM characters`).Scan(&charCount)
	if charCount != 5 {
		t.Fatalf("seed after legacy bootstrap: %d", charCount)
	}
}

func TestLearningDomainDownThenUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downup.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.SQL.Close()

	m2 := migrations.All()[1]
	if m2.Down == nil {
		t.Fatal("expected Down for learning_domain")
	}
	tx, err := store.SQL.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := m2.Down(tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("Down: %v", err)
	}
	if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version=2`); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var n int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='characters'`).Scan(&n)
	if n != 0 {
		t.Fatalf("characters still present after Down")
	}

	store.SQL.Close()
	store2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open after Down: %v", err)
	}
	defer store2.SQL.Close()
	_ = store2.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='characters'`).Scan(&n)
	if n != 1 {
		t.Fatalf("characters missing after re-Up")
	}
}

func TestHiragana5SeedMatchesRecognizeGlyphs(t *testing.T) {
	rec, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	defer rec.Close()

	seedGlyphs := Hiragana5Glyphs()
	if len(seedGlyphs) != 5 {
		t.Fatalf("seed glyphs=%d", len(seedGlyphs))
	}
	// Assess each glyph as supported target (unsupported would error).
	for _, g := range seedGlyphs {
		_, err := rec.Assess(g, nil, 300, 300)
		if err != nil && err != recognize.ErrUnsupportedTarget {
			t.Fatalf("Assess(%s): %v", g, err)
		}
		if err == recognize.ErrUnsupportedTarget {
			t.Fatalf("seed glyph %q not in recognize set", g)
		}
	}
}
