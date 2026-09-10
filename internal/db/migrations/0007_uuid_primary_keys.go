package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
)

// up0007UuidPrimaryKeys rebuilds surrogate INTEGER PK/FK tables as TEXT UUID keys,
// rewriting existing rows with freshly generated UUIDs. Curriculum natural keys and
// non-ID integer counters (board_rev, at_rev, …) are unchanged.
func up0007UuidPrimaryKeys(tx *sql.Tx) error {
	migrate0007Log("INFO", "[migrate.0007] start")

	if err := createUUIDTables(tx); err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=create_tables err=%v", err)
		return fmt.Errorf("create uuid tables: %w", err)
	}

	userMap, nUsers, err := copyUsers(tx)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=users err=%v", err)
		return err
	}

	nBoardState, err := copyUserBoardState(tx, userMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=user_board_state err=%v", err)
		return err
	}

	strokeMap, nStrokes, err := copyStrokes(tx, userMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=strokes err=%v", err)
		return err
	}

	nPoints, err := copyStrokePoints(tx, strokeMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=stroke_points err=%v", err)
		return err
	}

	nTombstones, err := copyStrokeOpTombstones(tx, userMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=stroke_op_tombstones err=%v", err)
		return err
	}

	nBoardOps, err := copyBoardOps(tx, userMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=board_ops err=%v", err)
		return err
	}

	attemptMap, nAttempts, err := copyPracticeAttempts(tx, userMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=practice_attempts err=%v", err)
		return err
	}

	attemptStrokeMap, nAttemptStrokes, err := copyAttemptStrokes(tx, attemptMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=attempt_strokes err=%v", err)
		return err
	}

	nAttemptPoints, err := copyAttemptStrokePoints(tx, attemptStrokeMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=attempt_stroke_points err=%v", err)
		return err
	}

	assessmentMap, nAssessments, err := copyAssessmentResults(tx, attemptMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=assessment_results err=%v", err)
		return err
	}

	nFeedback, err := copyAssessmentFeedback(tx, assessmentMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=assessment_feedback err=%v", err)
		return err
	}

	nProgress, err := copyUserCharacterProgress(tx, userMap, attemptMap)
	if err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=user_character_progress err=%v", err)
		return err
	}

	if err := swapUUIDTables(tx); err != nil {
		migrate0007Log("ERROR", "[migrate.0007] step=swap_tables err=%v", err)
		return err
	}

	migrate0007Log("INFO", "[migrate.0007] done users=%d board_state=%d strokes=%d stroke_points=%d tombstones=%d board_ops=%d attempts=%d attempt_strokes=%d attempt_points=%d assessments=%d feedback=%d progress=%d",
		nUsers, nBoardState, nStrokes, nPoints, nTombstones, nBoardOps, nAttempts, nAttemptStrokes, nAttemptPoints, nAssessments, nFeedback, nProgress)
	return nil
}

func createUUIDTables(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE users_new (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE strokes_new (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
		color TEXT NOT NULL,
		width INTEGER NOT NULL,
		started_at_unix_ms INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		op_id TEXT
	);
	CREATE TABLE stroke_points_new (
		id TEXT PRIMARY KEY,
		stroke_id TEXT NOT NULL REFERENCES strokes_new(id) ON DELETE CASCADE,
		x REAL NOT NULL,
		y REAL NOT NULL
	);
	CREATE TABLE user_board_state_new (
		user_id TEXT PRIMARY KEY REFERENCES users_new(id) ON DELETE CASCADE,
		board_rev INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE stroke_op_tombstones_new (
		user_id TEXT NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
		op_id TEXT NOT NULL,
		reason TEXT NOT NULL,
		at_rev INTEGER NOT NULL,
		PRIMARY KEY (user_id, op_id)
	);
	CREATE TABLE board_ops_new (
		user_id TEXT NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
		op_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		at_rev INTEGER NOT NULL,
		PRIMARY KEY (user_id, op_id)
	);
	CREATE TABLE practice_attempts_new (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
		character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
		lesson_id TEXT REFERENCES lessons(id) ON DELETE SET NULL,
		status TEXT NOT NULL,
		client_attempt_id TEXT,
		canvas_width INTEGER,
		canvas_height INTEGER,
		started_at TIMESTAMP NOT NULL,
		submitted_at TIMESTAMP,
		assessed_at TIMESTAMP,
		abandoned_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE attempt_strokes_new (
		id TEXT PRIMARY KEY,
		attempt_id TEXT NOT NULL REFERENCES practice_attempts_new(id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		color TEXT NOT NULL DEFAULT '#000000',
		width INTEGER NOT NULL DEFAULT 2,
		started_at_unix_ms INTEGER NOT NULL DEFAULT 0,
		UNIQUE(attempt_id, seq)
	);
	CREATE TABLE attempt_stroke_points_new (
		id TEXT PRIMARY KEY,
		attempt_stroke_id TEXT NOT NULL REFERENCES attempt_strokes_new(id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		x REAL NOT NULL,
		y REAL NOT NULL,
		UNIQUE(attempt_stroke_id, seq)
	);
	CREATE TABLE assessment_results_new (
		id TEXT PRIMARY KEY,
		attempt_id TEXT NOT NULL UNIQUE REFERENCES practice_attempts_new(id) ON DELETE CASCADE,
		pass INTEGER NOT NULL,
		score REAL NOT NULL,
		score_kind TEXT NOT NULL,
		assessor TEXT NOT NULL,
		set_id TEXT NOT NULL,
		reasons_json TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE assessment_feedback_new (
		id TEXT PRIMARY KEY,
		assessment_id TEXT NOT NULL REFERENCES assessment_results_new(id) ON DELETE CASCADE,
		rank INTEGER NOT NULL,
		code TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT '',
		UNIQUE(assessment_id, rank)
	);
	CREATE TABLE user_character_progress_new (
		user_id TEXT NOT NULL REFERENCES users_new(id) ON DELETE CASCADE,
		character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
		status TEXT NOT NULL,
		attempt_count INTEGER NOT NULL DEFAULT 0,
		pass_count INTEGER NOT NULL DEFAULT 0,
		last_attempt_id TEXT REFERENCES practice_attempts_new(id) ON DELETE SET NULL,
		last_passed_at TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		review_box INTEGER NOT NULL DEFAULT 0,
		due_at TIMESTAMP,
		last_reviewed_at TIMESTAMP,
		PRIMARY KEY (user_id, character_id)
	);
	`)
	return err
}

func copyUsers(tx *sql.Tx) (map[int64]string, int, error) {
	rows, err := tx.Query(`SELECT id, email, password_hash, created_at FROM users`)
	if err != nil {
		return nil, 0, fmt.Errorf("select users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[int64]string)
	n := 0
	for rows.Next() {
		var oldID int64
		var email, hash string
		var createdAt interface{}
		if err := rows.Scan(&oldID, &email, &hash, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan users: %w", err)
		}
		newID := uuid.NewString()
		m[oldID] = newID
		if _, err := tx.Exec(
			`INSERT INTO users_new(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
			newID, email, hash, createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("insert users_new old=%d: %w", oldID, err)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return m, n, nil
}

func copyUserBoardState(tx *sql.Tx, userMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT user_id, board_rev FROM user_board_state`)
	if err != nil {
		return 0, fmt.Errorf("select user_board_state: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldUID int64
		var boardRev int64
		if err := rows.Scan(&oldUID, &boardRev); err != nil {
			return 0, fmt.Errorf("scan user_board_state: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return 0, fmt.Errorf("user_board_state missing user map old=%d", oldUID)
		}
		if _, err := tx.Exec(
			`INSERT INTO user_board_state_new(user_id, board_rev) VALUES(?,?)`,
			newUID, boardRev,
		); err != nil {
			return 0, fmt.Errorf("insert user_board_state_new: %w", err)
		}
		n++
	}
	return n, rows.Err()
}

func copyStrokes(tx *sql.Tx, userMap map[int64]string) (map[int64]string, int, error) {
	rows, err := tx.Query(`SELECT id, user_id, color, width, started_at_unix_ms, created_at, op_id FROM strokes`)
	if err != nil {
		return nil, 0, fmt.Errorf("select strokes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[int64]string)
	n := 0
	for rows.Next() {
		var oldID, oldUID, width, startedAt int64
		var color string
		var createdAt interface{}
		var opID sql.NullString
		if err := rows.Scan(&oldID, &oldUID, &color, &width, &startedAt, &createdAt, &opID); err != nil {
			return nil, 0, fmt.Errorf("scan strokes: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return nil, 0, fmt.Errorf("strokes missing user map old_user=%d stroke=%d", oldUID, oldID)
		}
		newID := uuid.NewString()
		m[oldID] = newID
		var op any
		if opID.Valid {
			op = opID.String
		}
		if _, err := tx.Exec(
			`INSERT INTO strokes_new(id, user_id, color, width, started_at_unix_ms, created_at, op_id) VALUES(?,?,?,?,?,?,?)`,
			newID, newUID, color, width, startedAt, createdAt, op,
		); err != nil {
			return nil, 0, fmt.Errorf("insert strokes_new old=%d: %w", oldID, err)
		}
		n++
	}
	return m, n, rows.Err()
}

func copyStrokePoints(tx *sql.Tx, strokeMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT id, stroke_id, x, y FROM stroke_points`)
	if err != nil {
		return 0, fmt.Errorf("select stroke_points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldID, oldStrokeID int64
		var x, y float64
		if err := rows.Scan(&oldID, &oldStrokeID, &x, &y); err != nil {
			return 0, fmt.Errorf("scan stroke_points: %w", err)
		}
		newStrokeID, ok := strokeMap[oldStrokeID]
		if !ok {
			return 0, fmt.Errorf("stroke_points missing stroke map old_stroke=%d point=%d", oldStrokeID, oldID)
		}
		newID := uuid.NewString()
		if _, err := tx.Exec(
			`INSERT INTO stroke_points_new(id, stroke_id, x, y) VALUES(?,?,?,?)`,
			newID, newStrokeID, x, y,
		); err != nil {
			return 0, fmt.Errorf("insert stroke_points_new old=%d: %w", oldID, err)
		}
		n++
	}
	return n, rows.Err()
}

func copyStrokeOpTombstones(tx *sql.Tx, userMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT user_id, op_id, reason, at_rev FROM stroke_op_tombstones`)
	if err != nil {
		return 0, fmt.Errorf("select stroke_op_tombstones: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldUID, atRev int64
		var opID, reason string
		if err := rows.Scan(&oldUID, &opID, &reason, &atRev); err != nil {
			return 0, fmt.Errorf("scan stroke_op_tombstones: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return 0, fmt.Errorf("stroke_op_tombstones missing user map old=%d", oldUID)
		}
		if _, err := tx.Exec(
			`INSERT INTO stroke_op_tombstones_new(user_id, op_id, reason, at_rev) VALUES(?,?,?,?)`,
			newUID, opID, reason, atRev,
		); err != nil {
			return 0, fmt.Errorf("insert stroke_op_tombstones_new: %w", err)
		}
		n++
	}
	return n, rows.Err()
}

func copyBoardOps(tx *sql.Tx, userMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT user_id, op_id, kind, at_rev FROM board_ops`)
	if err != nil {
		return 0, fmt.Errorf("select board_ops: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldUID, atRev int64
		var opID, kind string
		if err := rows.Scan(&oldUID, &opID, &kind, &atRev); err != nil {
			return 0, fmt.Errorf("scan board_ops: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return 0, fmt.Errorf("board_ops missing user map old=%d", oldUID)
		}
		if _, err := tx.Exec(
			`INSERT INTO board_ops_new(user_id, op_id, kind, at_rev) VALUES(?,?,?,?)`,
			newUID, opID, kind, atRev,
		); err != nil {
			return 0, fmt.Errorf("insert board_ops_new: %w", err)
		}
		n++
	}
	return n, rows.Err()
}

func copyPracticeAttempts(tx *sql.Tx, userMap map[int64]string) (map[int64]string, int, error) {
	rows, err := tx.Query(`
		SELECT id, user_id, character_id, lesson_id, status, client_attempt_id,
			canvas_width, canvas_height, started_at, submitted_at, assessed_at, abandoned_at,
			created_at, updated_at
		FROM practice_attempts`)
	if err != nil {
		return nil, 0, fmt.Errorf("select practice_attempts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[int64]string)
	n := 0
	for rows.Next() {
		var oldID, oldUID int64
		var characterID, status string
		var lessonID, clientAttemptID sql.NullString
		var canvasW, canvasH sql.NullInt64
		var startedAt, submittedAt, assessedAt, abandonedAt, createdAt, updatedAt interface{}
		if err := rows.Scan(
			&oldID, &oldUID, &characterID, &lessonID, &status, &clientAttemptID,
			&canvasW, &canvasH, &startedAt, &submittedAt, &assessedAt, &abandonedAt,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan practice_attempts: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return nil, 0, fmt.Errorf("practice_attempts missing user map old_user=%d attempt=%d", oldUID, oldID)
		}
		newID := uuid.NewString()
		m[oldID] = newID
		if _, err := tx.Exec(`
			INSERT INTO practice_attempts_new(
				id, user_id, character_id, lesson_id, status, client_attempt_id,
				canvas_width, canvas_height, started_at, submitted_at, assessed_at, abandoned_at,
				created_at, updated_at
			) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			newID, newUID, characterID, nullStr(lessonID), status, nullStr(clientAttemptID),
			nullInt(canvasW), nullInt(canvasH), startedAt, submittedAt, assessedAt, abandonedAt,
			createdAt, updatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("insert practice_attempts_new old=%d: %w", oldID, err)
		}
		n++
	}
	return m, n, rows.Err()
}

func copyAttemptStrokes(tx *sql.Tx, attemptMap map[int64]string) (map[int64]string, int, error) {
	rows, err := tx.Query(`SELECT id, attempt_id, seq, color, width, started_at_unix_ms FROM attempt_strokes`)
	if err != nil {
		return nil, 0, fmt.Errorf("select attempt_strokes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[int64]string)
	n := 0
	for rows.Next() {
		var oldID, oldAttemptID, seq, width, startedAt int64
		var color string
		if err := rows.Scan(&oldID, &oldAttemptID, &seq, &color, &width, &startedAt); err != nil {
			return nil, 0, fmt.Errorf("scan attempt_strokes: %w", err)
		}
		newAttemptID, ok := attemptMap[oldAttemptID]
		if !ok {
			return nil, 0, fmt.Errorf("attempt_strokes missing attempt map old_attempt=%d stroke=%d", oldAttemptID, oldID)
		}
		newID := uuid.NewString()
		m[oldID] = newID
		if _, err := tx.Exec(
			`INSERT INTO attempt_strokes_new(id, attempt_id, seq, color, width, started_at_unix_ms) VALUES(?,?,?,?,?,?)`,
			newID, newAttemptID, seq, color, width, startedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("insert attempt_strokes_new old=%d: %w", oldID, err)
		}
		n++
	}
	return m, n, rows.Err()
}

func copyAttemptStrokePoints(tx *sql.Tx, attemptStrokeMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT id, attempt_stroke_id, seq, x, y FROM attempt_stroke_points`)
	if err != nil {
		return 0, fmt.Errorf("select attempt_stroke_points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldID, oldStrokeID, seq int64
		var x, y float64
		if err := rows.Scan(&oldID, &oldStrokeID, &seq, &x, &y); err != nil {
			return 0, fmt.Errorf("scan attempt_stroke_points: %w", err)
		}
		newStrokeID, ok := attemptStrokeMap[oldStrokeID]
		if !ok {
			return 0, fmt.Errorf("attempt_stroke_points missing stroke map old_stroke=%d point=%d", oldStrokeID, oldID)
		}
		newID := uuid.NewString()
		if _, err := tx.Exec(
			`INSERT INTO attempt_stroke_points_new(id, attempt_stroke_id, seq, x, y) VALUES(?,?,?,?,?)`,
			newID, newStrokeID, seq, x, y,
		); err != nil {
			return 0, fmt.Errorf("insert attempt_stroke_points_new old=%d: %w", oldID, err)
		}
		n++
	}
	return n, rows.Err()
}

func copyAssessmentResults(tx *sql.Tx, attemptMap map[int64]string) (map[int64]string, int, error) {
	rows, err := tx.Query(`
		SELECT id, attempt_id, pass, score, score_kind, assessor, set_id, reasons_json, created_at
		FROM assessment_results`)
	if err != nil {
		return nil, 0, fmt.Errorf("select assessment_results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[int64]string)
	n := 0
	for rows.Next() {
		var oldID, oldAttemptID, pass int64
		var score float64
		var scoreKind, assessor, setID, reasonsJSON string
		var createdAt interface{}
		if err := rows.Scan(&oldID, &oldAttemptID, &pass, &score, &scoreKind, &assessor, &setID, &reasonsJSON, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan assessment_results: %w", err)
		}
		newAttemptID, ok := attemptMap[oldAttemptID]
		if !ok {
			return nil, 0, fmt.Errorf("assessment_results missing attempt map old_attempt=%d assessment=%d", oldAttemptID, oldID)
		}
		newID := uuid.NewString()
		m[oldID] = newID
		if _, err := tx.Exec(
			`INSERT INTO assessment_results_new(id, attempt_id, pass, score, score_kind, assessor, set_id, reasons_json, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
			newID, newAttemptID, pass, score, scoreKind, assessor, setID, reasonsJSON, createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("insert assessment_results_new old=%d: %w", oldID, err)
		}
		n++
	}
	return m, n, rows.Err()
}

func copyAssessmentFeedback(tx *sql.Tx, assessmentMap map[int64]string) (int, error) {
	rows, err := tx.Query(`SELECT id, assessment_id, rank, code, message FROM assessment_feedback`)
	if err != nil {
		return 0, fmt.Errorf("select assessment_feedback: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldID, oldAssessmentID, rank int64
		var code, message string
		if err := rows.Scan(&oldID, &oldAssessmentID, &rank, &code, &message); err != nil {
			return 0, fmt.Errorf("scan assessment_feedback: %w", err)
		}
		newAssessmentID, ok := assessmentMap[oldAssessmentID]
		if !ok {
			return 0, fmt.Errorf("assessment_feedback missing assessment map old_assessment=%d feedback=%d", oldAssessmentID, oldID)
		}
		newID := uuid.NewString()
		if _, err := tx.Exec(
			`INSERT INTO assessment_feedback_new(id, assessment_id, rank, code, message) VALUES(?,?,?,?,?)`,
			newID, newAssessmentID, rank, code, message,
		); err != nil {
			return 0, fmt.Errorf("insert assessment_feedback_new old=%d: %w", oldID, err)
		}
		n++
	}
	return n, rows.Err()
}

func copyUserCharacterProgress(tx *sql.Tx, userMap, attemptMap map[int64]string) (int, error) {
	rows, err := tx.Query(`
		SELECT user_id, character_id, status, attempt_count, pass_count, last_attempt_id,
			last_passed_at, updated_at, review_box, due_at, last_reviewed_at
		FROM user_character_progress`)
	if err != nil {
		return 0, fmt.Errorf("select user_character_progress: %w", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		var oldUID, attemptCount, passCount, reviewBox int64
		var characterID, status string
		var lastAttemptID sql.NullInt64
		var lastPassedAt, updatedAt, dueAt, lastReviewedAt interface{}
		if err := rows.Scan(
			&oldUID, &characterID, &status, &attemptCount, &passCount, &lastAttemptID,
			&lastPassedAt, &updatedAt, &reviewBox, &dueAt, &lastReviewedAt,
		); err != nil {
			return 0, fmt.Errorf("scan user_character_progress: %w", err)
		}
		newUID, ok := userMap[oldUID]
		if !ok {
			return 0, fmt.Errorf("user_character_progress missing user map old=%d", oldUID)
		}
		var newLastAttempt any
		if lastAttemptID.Valid {
			mapped, ok := attemptMap[lastAttemptID.Int64]
			if !ok {
				return 0, fmt.Errorf("user_character_progress missing attempt map last_attempt=%d user=%d", lastAttemptID.Int64, oldUID)
			}
			newLastAttempt = mapped
		}
		if _, err := tx.Exec(`
			INSERT INTO user_character_progress_new(
				user_id, character_id, status, attempt_count, pass_count, last_attempt_id,
				last_passed_at, updated_at, review_box, due_at, last_reviewed_at
			) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			newUID, characterID, status, attemptCount, passCount, newLastAttempt,
			lastPassedAt, updatedAt, reviewBox, dueAt, lastReviewedAt,
		); err != nil {
			return 0, fmt.Errorf("insert user_character_progress_new: %w", err)
		}
		n++
	}
	return n, rows.Err()
}

func swapUUIDTables(tx *sql.Tx) error {
	// Drop children before parents so FK checks succeed with foreign_keys=ON.
	_, err := tx.Exec(`
	DROP TABLE IF EXISTS stroke_points;
	DROP TABLE IF EXISTS strokes;
	DROP TABLE IF EXISTS stroke_op_tombstones;
	DROP TABLE IF EXISTS board_ops;
	DROP TABLE IF EXISTS user_board_state;
	DROP TABLE IF EXISTS assessment_feedback;
	DROP TABLE IF EXISTS assessment_results;
	DROP TABLE IF EXISTS attempt_stroke_points;
	DROP TABLE IF EXISTS attempt_strokes;
	DROP TABLE IF EXISTS user_character_progress;
	DROP TABLE IF EXISTS practice_attempts;
	DROP TABLE IF EXISTS users;

	ALTER TABLE users_new RENAME TO users;
	ALTER TABLE strokes_new RENAME TO strokes;
	ALTER TABLE stroke_points_new RENAME TO stroke_points;
	ALTER TABLE user_board_state_new RENAME TO user_board_state;
	ALTER TABLE stroke_op_tombstones_new RENAME TO stroke_op_tombstones;
	ALTER TABLE board_ops_new RENAME TO board_ops;
	ALTER TABLE practice_attempts_new RENAME TO practice_attempts;
	ALTER TABLE attempt_strokes_new RENAME TO attempt_strokes;
	ALTER TABLE attempt_stroke_points_new RENAME TO attempt_stroke_points;
	ALTER TABLE assessment_results_new RENAME TO assessment_results;
	ALTER TABLE assessment_feedback_new RENAME TO assessment_feedback;
	ALTER TABLE user_character_progress_new RENAME TO user_character_progress;

	CREATE INDEX IF NOT EXISTS idx_strokes_user ON strokes(user_id);
	CREATE INDEX IF NOT EXISTS idx_stroke_points_stroke ON stroke_points(stroke_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_strokes_user_op ON strokes(user_id, op_id) WHERE op_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_attempts_user_started ON practice_attempts(user_id, started_at DESC);
	CREATE INDEX IF NOT EXISTS idx_attempts_user_char ON practice_attempts(user_id, character_id, started_at DESC);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_attempts_user_client ON practice_attempts(user_id, client_attempt_id) WHERE client_attempt_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_attempt_stroke_points_stroke ON attempt_stroke_points(attempt_stroke_id);
	CREATE INDEX IF NOT EXISTS idx_progress_user_due ON user_character_progress(user_id, due_at);
	`)
	return err
}

func nullStr(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}

func nullInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func migrate0007Log(level, format string, args ...interface{}) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error":
		if level == "DEBUG" || level == "INFO" || level == "WARN" {
			return
		}
	case "warn":
		if level == "DEBUG" || level == "INFO" {
			return
		}
	case "info":
		if level == "DEBUG" {
			return
		}
	}
	log.Printf(level+" "+format, args...)
}
