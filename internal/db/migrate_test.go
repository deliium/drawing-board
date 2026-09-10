package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/curriculum"
	"github.com/deliium/drawing-board/internal/db/migrations"
	"github.com/deliium/drawing-board/internal/learn"
	"github.com/deliium/drawing-board/internal/recognize"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateFreshOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()

	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if ver != migrations.LatestVersion() {
		t.Fatalf("version=%d want %d", ver, migrations.LatestVersion())
	}
	if ver < 5 {
		t.Fatalf("expected review_schedule migration version>=5 got %d", ver)
	}

	var hasReviewBox int
	if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('user_character_progress') WHERE name='review_box'`).Scan(&hasReviewBox); err != nil || hasReviewBox != 1 {
		t.Fatalf("review_box column missing: n=%d err=%v", hasReviewBox, err)
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

	var rom, contentVer, desc string
	if err := store.SQL.QueryRow(`
		SELECT COALESCE(romanization,''), COALESCE(content_version,''), COALESCE(description_en,'')
		FROM characters WHERE id='hira:あ'`).Scan(&rom, &contentVer, &desc); err != nil {
		t.Fatalf("pedagogy fields: %v", err)
	}
	if rom != "a" || contentVer == "" || desc == "" {
		t.Fatalf("pedagogy incomplete romanization=%q contentVersion=%q desc empty=%t", rom, contentVer, desc == "")
	}
	t.Logf("seed contentVersion=%s", contentVer)

	// Re-open is idempotent.
	_ = store.SQL.Close()
	store2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer func() { _ = store2.SQL.Close() }()
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
	defer func() { _ = store.SQL.Close() }()

	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if ver != migrations.LatestVersion() {
		t.Fatalf("version=%d want %d", ver, migrations.LatestVersion())
	}

	u, err := store.GetUserByEmail("legacy@example.com")
	if err != nil || u == nil {
		t.Fatalf("GetUserByEmail: %v u=%v", err, u)
	}
	if u.ID == "" {
		t.Fatalf("expected remapped UUID user id, got empty")
	}
	strokes, err := store.ListStrokesByUser(u.ID)
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
	if !strings.Contains(strokes[0].ID, "-") {
		t.Fatalf("expected UUID stroke id, got %q", strokes[0].ID)
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
	defer func() { _ = store.SQL.Close() }()

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
	// Drop learning_domain and any later versions that depend on those tables (e.g. pedagogy).
	if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version >= 2`); err != nil {
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

	_ = store.SQL.Close()
	store2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open after Down: %v", err)
	}
	defer func() { _ = store2.SQL.Close() }()
	_ = store2.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='characters'`).Scan(&n)
	if n != 1 {
		t.Fatalf("characters missing after re-Up")
	}
}

func TestMigrateFailClosedBadUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fail-closed.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()

	before, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	t.Logf("[db.migrate] before fail-closed version=%d", before)

	bad := migrations.Migration{
		Version: before + 100,
		Name:    "intentional_fail",
		Up: func(tx *sql.Tx) error {
			return sql.ErrConnDone
		},
	}
	if err := applyMigration(store.SQL, bad); err == nil {
		t.Fatal("expected applyMigration to fail")
	}

	after, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion after: %v", err)
	}
	t.Logf("[db.migrate] after fail-closed version=%d", after)
	if after != before {
		t.Fatalf("fail-closed violated: version advanced %d→%d", before, after)
	}
	var n int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version=?`, bad.Version).Scan(&n)
	if n != 0 {
		t.Fatalf("failed migration recorded in schema_migrations")
	}
}

func TestSeedHiragana5IdempotentAndBoardAttemptIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed-iso.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()

	uid, err := store.CreateUser("iso@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	_, err = store.ApplyStrokeCreate(uid, 0, "op-board-1", "#000000", 2, 1, []StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("board stroke: %v", err)
	}

	ls := NewLearnStore(store)
	ctx := context.Background()
	att, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID:      uid,
		CharacterID: "hira:あ",
		LessonID:    "lesson:hiragana5",
	})
	if err != nil {
		t.Fatalf("attempt create: %v", err)
	}
	t.Logf("[db.migrate] attempt id=%s before board clear", att.ID)

	if err := SeedHiragana5(store); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	var charCount int
	if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM characters WHERE set_id=?`, recognize.SetIDHiragana5).Scan(&charCount); err != nil || charCount != 5 {
		t.Fatalf("seed characters after re-seed=%d err=%v", charCount, err)
	}

	if _, err := store.ApplyClear(uid, 1, "op-clear-iso"); err != nil {
		t.Fatalf("ApplyClear: %v", err)
	}
	t.Logf("[db.migrate] cleared board")
	strokes, err := store.ListStrokesByUser(uid)
	if err != nil {
		t.Fatalf("ListStrokes: %v", err)
	}
	if len(strokes) != 0 {
		t.Fatalf("board strokes after clear=%d want 0", len(strokes))
	}

	got, err := ls.Attempts().Get(ctx, uid, att.ID)
	if err != nil {
		t.Fatalf("attempt after board clear: %v", err)
	}
	if got == nil || got.ID != att.ID {
		t.Fatalf("attempt missing after board clear")
	}
	var attemptStrokeTables int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name IN ('attempt_strokes','attempt_stroke_points')`).Scan(&attemptStrokeTables)
	if attemptStrokeTables != 2 {
		t.Fatalf("attempt stroke tables missing after board clear")
	}
}

func TestMigrateUpgradeFromVersion2Fixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "from-v2.db")
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("fk: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	tx, err := raw.Begin()
	if err != nil {
		t.Fatal(err)
	}
	all := migrations.All()
	for _, m := range all {
		if m.Version > 2 {
			break
		}
		if err := m.Up(tx); err != nil {
			_ = tx.Rollback()
			t.Fatalf("Up v%d: %v", m.Version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, name) VALUES(?,?)`, m.Version, m.Name); err != nil {
			_ = tx.Rollback()
			t.Fatalf("stamp v%d: %v", m.Version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open upgrade: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()
	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	t.Logf("[db.migrate] upgraded from v2 → %d", ver)
	if ver != migrations.LatestVersion() {
		t.Fatalf("version=%d want %d", ver, migrations.LatestVersion())
	}
	var hasGuidance int
	if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('characters') WHERE name='guidance_en'`).Scan(&hasGuidance); err != nil || hasGuidance != 1 {
		t.Fatalf("guidance_en missing after upgrade: n=%d err=%v", hasGuidance, err)
	}
}

func TestHiragana5SeedMatchesRecognizeGlyphs(t *testing.T) {
	rec, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	defer func() { _ = rec.Close() }()

	seedGlyphs := Hiragana5Glyphs()
	if len(seedGlyphs) != 5 {
		t.Fatalf("seed glyphs=%d", len(seedGlyphs))
	}
	if rec.Version() != Hiragana5ContentVersion() {
		t.Fatalf("contentVersion seed=%q recognize=%q", Hiragana5ContentVersion(), rec.Version())
	}
	for _, g := range seedGlyphs {
		_, err := rec.Assess(g, nil, 300, 300)
		if err != nil && err != recognize.ErrUnsupportedTarget {
			t.Fatalf("Assess(%s): %v", g, err)
		}
		if err == recognize.ErrUnsupportedTarget {
			t.Fatalf("seed glyph %q not in recognize set", g)
		}
	}

	store, err := Open(filepath.Join(t.TempDir(), "align.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()
	ls := NewLearnStore(store)
	chars, err := ls.Characters().ListBySet(context.Background(), recognize.SetIDHiragana5)
	if err != nil {
		t.Fatalf("ListBySet: %v", err)
	}
	pack, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	for _, c := range chars {
		if c.Romanization == "" || c.ContentVersion == "" {
			t.Fatalf("character %s missing pedagogy", c.ID)
		}
		n := len(pack.Strokes[c.Glyph])
		if n != c.StrokeCount {
			t.Fatalf("glyph %s strokeCount seed=%d pack=%d", c.Glyph, c.StrokeCount, n)
		}
	}
}
