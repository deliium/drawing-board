package db

import (
	"database/sql"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/deliium/drawing-board/internal/db/migrations"
	"github.com/deliium/drawing-board/internal/ids"
	_ "github.com/mattn/go-sqlite3"
)

var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// TestMigrate0007UuidRewriteIntegrity seeds a v6-shaped DB with board + learning rows,
// Open→v7, and asserts counts, UUID forms, FK joins, board_rev, and curriculum IDs.
func TestMigrate0007UuidRewriteIntegrity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v6-rewrite.db")
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("fk: %v", err)
	}

	tx, err := raw.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	for _, m := range migrations.All() {
		if m.Version > 6 {
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
	// Minimal curriculum rows so attempt FKs resolve (seed runs on Open later too).
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO characters(id, set_id, glyph, romanization, stroke_count, sort_key, status)
		VALUES
			('hira:あ','hiragana5','あ','a',3,1,'active'),
			('hira:い','hiragana5','い','i',2,2,'active');
		INSERT OR IGNORE INTO lessons(id, code, title, set_id, sort_order, status)
		VALUES('lesson:hiragana5','hiragana5','Hiragana あ行','hiragana5',1,'published');
	`); err != nil {
		_ = tx.Rollback()
		t.Fatalf("curriculum seed: %v", err)
	}

	res, err := tx.Exec(`INSERT INTO users(email, password_hash) VALUES('uuid-mig@example.com','hash')`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("user: %v", err)
	}
	uid, _ := res.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO user_board_state(user_id, board_rev) VALUES(?, 7)`, uid); err != nil {
		_ = tx.Rollback()
		t.Fatalf("board_state: %v", err)
	}
	sres, err := tx.Exec(`INSERT INTO strokes(user_id, color, width, started_at_unix_ms, op_id) VALUES(?,?,?,?,?)`, uid, "#111111", 3, 100, "op-keep")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("stroke: %v", err)
	}
	sid, _ := sres.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO stroke_points(stroke_id, x, y) VALUES(?,?,?)`, sid, 1.5, 2.5); err != nil {
		_ = tx.Rollback()
		t.Fatalf("point: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO board_ops(user_id, op_id, kind, at_rev) VALUES(?,?,?,?)`, uid, "op-keep", "create", 1); err != nil {
		_ = tx.Rollback()
		t.Fatalf("board_ops: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO stroke_op_tombstones(user_id, op_id, reason, at_rev) VALUES(?,?,?,?)`, uid, "op-tomb", "delete", 3); err != nil {
		_ = tx.Rollback()
		t.Fatalf("tombstone: %v", err)
	}

	ares, err := tx.Exec(`
		INSERT INTO practice_attempts(user_id, character_id, lesson_id, status, client_attempt_id, canvas_width, canvas_height, started_at)
		VALUES(?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		uid, "hira:あ", "lesson:hiragana5", "assessed", "client-1", 300, 300)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("attempt: %v", err)
	}
	aid, _ := ares.LastInsertId()
	asres, err := tx.Exec(`INSERT INTO attempt_strokes(attempt_id, seq, color, width, started_at_unix_ms) VALUES(?,?,?,?,?)`, aid, 0, "#000", 2, 1)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("attempt_stroke: %v", err)
	}
	asid, _ := asres.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO attempt_stroke_points(attempt_stroke_id, seq, x, y) VALUES(?,?,?,?)`, asid, 0, 9, 8); err != nil {
		_ = tx.Rollback()
		t.Fatalf("attempt_point: %v", err)
	}
	assesRes, err := tx.Exec(`
		INSERT INTO assessment_results(attempt_id, pass, score, score_kind, assessor, set_id, reasons_json)
		VALUES(?,?,?,?,?,?,?)`, aid, 1, 0.9, "match", "target_compare", "hiragana5", "[]")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("assessment: %v", err)
	}
	assessID, _ := assesRes.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO assessment_feedback(assessment_id, rank, code, message) VALUES(?,?,?,?)`, assessID, 1, "ok", "good"); err != nil {
		_ = tx.Rollback()
		t.Fatalf("feedback: %v", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO user_character_progress(user_id, character_id, status, attempt_count, pass_count, last_attempt_id, review_box)
		VALUES(?,?,?,?,?,?,?)`, uid, "hira:あ", "passed", 1, 1, aid, 1); err != nil {
		_ = tx.Rollback()
		t.Fatalf("progress: %v", err)
	}
	// Progress row with NULL last_attempt_id (edge copier path).
	if _, err := tx.Exec(`
		INSERT INTO user_character_progress(user_id, character_id, status, attempt_count, pass_count, last_attempt_id, review_box)
		VALUES(?,?,?,?,?,?,?)`, uid, "hira:い", "not_started", 0, 0, nil, 0); err != nil {
		_ = tx.Rollback()
		t.Fatalf("progress null last_attempt: %v", err)
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
	if err != nil || ver != 7 {
		t.Fatalf("version=%d err=%v want 7", ver, err)
	}

	assertCount := func(q string, want int) {
		t.Helper()
		var n int
		if err := store.SQL.QueryRow(q).Scan(&n); err != nil || n != want {
			t.Fatalf("%s → n=%d err=%v want %d", q, n, err, want)
		}
	}
	assertCount(`SELECT COUNT(1) FROM users`, 1)
	assertCount(`SELECT COUNT(1) FROM strokes`, 1)
	assertCount(`SELECT COUNT(1) FROM stroke_points`, 1)
	assertCount(`SELECT COUNT(1) FROM stroke_op_tombstones`, 1)
	assertCount(`SELECT COUNT(1) FROM board_ops`, 1)
	assertCount(`SELECT COUNT(1) FROM practice_attempts`, 1)
	assertCount(`SELECT COUNT(1) FROM attempt_strokes`, 1)
	assertCount(`SELECT COUNT(1) FROM attempt_stroke_points`, 1)
	assertCount(`SELECT COUNT(1) FROM assessment_results`, 1)
	assertCount(`SELECT COUNT(1) FROM assessment_feedback`, 1)
	assertCount(`SELECT COUNT(1) FROM user_character_progress`, 2)

	u, err := store.GetUserByEmail("uuid-mig@example.com")
	if err != nil || u == nil || !ids.Valid(u.ID) {
		t.Fatalf("user after rewrite: %+v err=%v", u, err)
	}

	var boardRev int64
	if err := store.SQL.QueryRow(`SELECT board_rev FROM user_board_state WHERE user_id=?`, u.ID).Scan(&boardRev); err != nil || boardRev != 7 {
		t.Fatalf("board_rev=%d err=%v want 7", boardRev, err)
	}

	strokes, err := store.ListStrokesByUser(u.ID)
	if err != nil || len(strokes) != 1 || !ids.Valid(strokes[0].ID) {
		t.Fatalf("strokes=%+v err=%v", strokes, err)
	}
	if strokes[0].OpID != "op-keep" || len(strokes[0].Points) != 1 {
		t.Fatalf("stroke payload: %+v", strokes[0])
	}

	var attemptID, lastAttemptID, charID, lessonID string
	if err := store.SQL.QueryRow(`
		SELECT a.id, p.last_attempt_id, a.character_id, a.lesson_id
		FROM practice_attempts a
		JOIN user_character_progress p ON p.last_attempt_id = a.id AND p.user_id = a.user_id
		WHERE a.user_id=?`, u.ID).Scan(&attemptID, &lastAttemptID, &charID, &lessonID); err != nil {
		t.Fatalf("attempt/progress join: %v", err)
	}
	if !ids.Valid(attemptID) || attemptID != lastAttemptID {
		t.Fatalf("attemptID=%q last=%q", attemptID, lastAttemptID)
	}
	if charID != "hira:あ" || lessonID != "lesson:hiragana5" {
		t.Fatalf("curriculum ids changed: char=%q lesson=%q", charID, lessonID)
	}

	var tombOp, tombReason string
	var tombRev int64
	if err := store.SQL.QueryRow(`
		SELECT op_id, reason, at_rev FROM stroke_op_tombstones WHERE user_id=?`, u.ID).Scan(&tombOp, &tombReason, &tombRev); err != nil {
		t.Fatalf("tombstone: %v", err)
	}
	if tombOp != "op-tomb" || tombReason != "delete" || tombRev != 3 {
		t.Fatalf("tombstone payload: op=%q reason=%q rev=%d", tombOp, tombReason, tombRev)
	}

	var nullLast sql.NullString
	if err := store.SQL.QueryRow(`
		SELECT last_attempt_id FROM user_character_progress WHERE user_id=? AND character_id='hira:い'`, u.ID).Scan(&nullLast); err != nil {
		t.Fatalf("null last_attempt progress: %v", err)
	}
	if nullLast.Valid {
		t.Fatalf("expected NULL last_attempt_id, got %q", nullLast.String)
	}

	for _, idx := range []string{
		"idx_strokes_user",
		"idx_stroke_points_stroke",
		"idx_strokes_user_op",
		"idx_attempts_user_started",
		"idx_attempts_user_char",
		"idx_attempts_user_client",
		"idx_attempt_stroke_points_stroke",
		"idx_progress_user_due",
	} {
		var n int
		if err := store.SQL.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil || n != 1 {
			t.Fatalf("index %s present=%d err=%v", idx, n, err)
		}
	}

	var feedbackN int
	if err := store.SQL.QueryRow(`
		SELECT COUNT(1) FROM assessment_feedback f
		JOIN assessment_results r ON r.id = f.assessment_id
		WHERE r.attempt_id=?`, attemptID).Scan(&feedbackN); err != nil || feedbackN != 1 {
		t.Fatalf("assessment chain feedback=%d err=%v", feedbackN, err)
	}

	// PK type sanity: users.id is TEXT.
	var idType string
	if err := store.SQL.QueryRow(`SELECT typeof(id) FROM users LIMIT 1`).Scan(&idType); err != nil || idType != "text" {
		t.Fatalf("users.id typeof=%q err=%v", idType, err)
	}
	if !uuidRE.MatchString(u.ID) {
		t.Fatalf("user id not UUID form: %q", u.ID)
	}

	_ = store.SQL.Close()
	store2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer func() { _ = store2.SQL.Close() }()
	ver2, _ := store2.SchemaVersion()
	if ver2 != 7 {
		t.Fatalf("re-open version=%d", ver2)
	}
	t.Logf("[migrate.0007] rewrite integrity ok userID=%s attemptID=%s boardRev=%d", u.ID, attemptID, boardRev)
}
