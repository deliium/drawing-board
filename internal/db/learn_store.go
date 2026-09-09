package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/deliium/drawing-board/internal/learn"
	"github.com/deliium/drawing-board/internal/limits"
)

// LearnStore implements learn repository interfaces against SQLite.
type LearnStore struct {
	s *Store
}

// NewLearnStore wraps a Store for learning-domain repos.
func NewLearnStore(s *Store) *LearnStore {
	return &LearnStore{s: s}
}

func learnLog(level, format string, args ...interface{}) {
	dbLog(level, format, args...)
}

func (ls *LearnStore) ListBySet(ctx context.Context, setID string) ([]learn.Character, error) {
	_ = ctx
	rows, err := ls.s.SQL.Query(`
		SELECT id, set_id, glyph, COALESCE(romanization, ''), stroke_count, sort_key, status,
			COALESCE(description_en, ''), COALESCE(description_ja, ''), COALESCE(pronunciation_json, '{}'),
			COALESCE(example_word, ''), COALESCE(example_romanization, ''), COALESCE(example_meaning_en, ''),
			COALESCE(example_meaning_ja, ''),
			COALESCE(content_version, ''), COALESCE(trace_ref, ''),
			created_at, updated_at
		FROM characters WHERE set_id = ? ORDER BY sort_key
	`, setID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []learn.Character
	for rows.Next() {
		var c learn.Character
		if err := scanCharacter(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (ls *LearnStore) getCharacter(ctx context.Context, id string) (*learn.Character, error) {
	_ = ctx
	row := ls.s.SQL.QueryRow(`
		SELECT id, set_id, glyph, COALESCE(romanization, ''), stroke_count, sort_key, status,
			COALESCE(description_en, ''), COALESCE(description_ja, ''), COALESCE(pronunciation_json, '{}'),
			COALESCE(example_word, ''), COALESCE(example_romanization, ''), COALESCE(example_meaning_en, ''),
			COALESCE(example_meaning_ja, ''),
			COALESCE(content_version, ''), COALESCE(trace_ref, ''),
			created_at, updated_at
		FROM characters WHERE id = ?
	`, id)
	var c learn.Character
	if err := scanCharacter(row, &c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, learn.ErrNotFound
		}
		return nil, err
	}
	learnLog("DEBUG", "[learn.CharacterRepo.Get] id=%s contentVersion=%s", c.ID, c.ContentVersion)
	return &c, nil
}

type characterScanner interface {
	Scan(dest ...any) error
}

func scanCharacter(row characterScanner, c *learn.Character) error {
	return row.Scan(
		&c.ID, &c.SetID, &c.Glyph, &c.Romanization, &c.StrokeCount, &c.SortKey, &c.Status,
		&c.DescriptionEn, &c.DescriptionJa, &c.PronunciationJSON,
		&c.ExampleWord, &c.ExampleRomanization, &c.ExampleMeaningEn, &c.ExampleMeaningJa,
		&c.ContentVersion, &c.TraceRef,
		&c.CreatedAt, &c.UpdatedAt,
	)
}

func (ls *LearnStore) GetPublished(ctx context.Context, id string) (*learn.Lesson, error) {
	_ = ctx
	row := ls.s.SQL.QueryRow(`
		SELECT id, code, title, COALESCE(title_ja, ''), set_id, sort_order, status, created_at, updated_at
		FROM lessons WHERE id = ? AND status = ?
	`, id, learn.LessonStatusPublished)
	var l learn.Lesson
	if err := row.Scan(&l.ID, &l.Code, &l.Title, &l.TitleJa, &l.SetID, &l.SortOrder, &l.Status, &l.CreatedAt, &l.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, learn.ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (ls *LearnStore) ListCharacters(ctx context.Context, lessonID string) ([]learn.LessonCharacter, error) {
	_ = ctx
	rows, err := ls.s.SQL.Query(`
		SELECT lc.lesson_id, lc.character_id, c.glyph, lc.position
		FROM lesson_characters lc
		JOIN characters c ON c.id = lc.character_id
		WHERE lc.lesson_id = ?
		ORDER BY lc.position
	`, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []learn.LessonCharacter
	for rows.Next() {
		var lc learn.LessonCharacter
		if err := rows.Scan(&lc.LessonID, &lc.CharacterID, &lc.Glyph, &lc.Position); err != nil {
			return nil, err
		}
		out = append(out, lc)
	}
	return out, rows.Err()
}

func (ls *LearnStore) CreateDraft(ctx context.Context, in learn.CreateDraft) (learn.Attempt, error) {
	_ = ctx
	learnLog("DEBUG", "[learn.AttemptRepo.CreateDraft] userID=%d characterID=%s", in.UserID, in.CharacterID)
	if in.UserID <= 0 || in.CharacterID == "" {
		return learn.Attempt{}, learn.ErrInvalidInput
	}
	if in.ClientAttemptID != "" && len(in.ClientAttemptID) > limits.MaxOpIDLen {
		return learn.Attempt{}, learn.ErrInvalidInput
	}

	tx, err := ls.s.beginImmediate()
	if err != nil {
		return learn.Attempt{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var charExists int
	if err := tx.QueryRow(`SELECT 1 FROM characters WHERE id = ?`, in.CharacterID).Scan(&charExists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return learn.Attempt{}, learn.ErrNotFound
		}
		return learn.Attempt{}, err
	}

	now := time.Now().UTC()
	var lessonArg any
	if in.LessonID != "" {
		lessonArg = in.LessonID
	}
	var clientArg any
	if in.ClientAttemptID != "" {
		clientArg = in.ClientAttemptID
	}

	res, err := tx.Exec(`
		INSERT INTO practice_attempts(user_id, character_id, lesson_id, status, client_attempt_id, started_at, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
	`, in.UserID, in.CharacterID, lessonArg, learn.AttemptStatusDraft, clientArg, now, now, now)
	if err != nil {
		if isUniqueConstraintErr(err) {
			_ = tx.Rollback()
			existing, gerr := ls.getByClientAttemptID(in.UserID, in.ClientAttemptID)
			if gerr != nil {
				return learn.Attempt{}, gerr
			}
			if existing.CharacterID != in.CharacterID || existing.LessonID != in.LessonID {
				learnLog("WARN", "[learn.AttemptRepo.CreateDraft] conflict mismatch userID=%d clientAttemptID=%s", in.UserID, in.ClientAttemptID)
				return learn.Attempt{}, learn.ErrConflict
			}
			learnLog("INFO", "[learn.AttemptRepo.CreateDraft] idempotent replay userID=%d attemptID=%d clientAttemptID=%s",
				in.UserID, existing.ID, in.ClientAttemptID)
			return existing, nil
		}
		return learn.Attempt{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return learn.Attempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return learn.Attempt{}, err
	}
	return ls.getAttempt(in.UserID, id)
}

func (ls *LearnStore) getAttempt(userID, attemptID int64) (learn.Attempt, error) {
	row := ls.s.SQL.QueryRow(`
		SELECT id, user_id, character_id, lesson_id, status, client_attempt_id,
			canvas_width, canvas_height, started_at, submitted_at, assessed_at, abandoned_at, created_at, updated_at
		FROM practice_attempts WHERE id = ? AND user_id = ?
	`, attemptID, userID)
	return scanAttempt(row)
}

func (ls *LearnStore) getByClientAttemptID(userID int64, clientAttemptID string) (learn.Attempt, error) {
	if clientAttemptID == "" {
		return learn.Attempt{}, learn.ErrInvalidInput
	}
	row := ls.s.SQL.QueryRow(`
		SELECT id, user_id, character_id, lesson_id, status, client_attempt_id,
			canvas_width, canvas_height, started_at, submitted_at, assessed_at, abandoned_at, created_at, updated_at
		FROM practice_attempts WHERE user_id = ? AND client_attempt_id = ?
	`, userID, clientAttemptID)
	return scanAttempt(row)
}

// GetByClientAttemptID returns an attempt owned by userID for the given clientAttemptID.
func (ls *LearnStore) GetByClientAttemptID(ctx context.Context, userID int64, clientAttemptID string) (*learn.Attempt, error) {
	_ = ctx
	a, err := ls.getByClientAttemptID(userID, clientAttemptID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListStrokes returns ordered stroke geometry for a submitted/assessed attempt (ownership-checked).
func (ls *LearnStore) ListStrokes(ctx context.Context, userID, attemptID int64) ([]learn.StrokeInput, int, int, error) {
	_ = ctx
	a, err := ls.getAttempt(userID, attemptID)
	if err != nil {
		return nil, 0, 0, err
	}
	if a.Status != learn.AttemptStatusSubmitted && a.Status != learn.AttemptStatusAssessed {
		learnLog("WARN", "[learn.AttemptRepo.ListStrokes] invalid_status attemptID=%d status=%s", attemptID, a.Status)
		return nil, 0, 0, learn.ErrInvalidStatus
	}
	if a.CanvasWidth == nil || a.CanvasHeight == nil {
		return nil, 0, 0, learn.ErrInvalidStatus
	}

	rows, err := ls.s.SQL.Query(`
		SELECT id, color, width, started_at_unix_ms
		FROM attempt_strokes WHERE attempt_id = ? ORDER BY seq ASC
	`, attemptID)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var strokes []learn.StrokeInput
	for rows.Next() {
		var strokeID int64
		var st learn.StrokeInput
		if err := rows.Scan(&strokeID, &st.Color, &st.Width, &st.StartedAtUnixMs); err != nil {
			return nil, 0, 0, err
		}
		pts, err := ls.loadAttemptStrokePoints(strokeID)
		if err != nil {
			return nil, 0, 0, err
		}
		st.Points = pts
		strokes = append(strokes, st)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}
	learnLog("DEBUG", "[learn.AttemptRepo.ListStrokes] attemptID=%d strokeCount=%d", attemptID, len(strokes))
	return strokes, *a.CanvasWidth, *a.CanvasHeight, nil
}

func (ls *LearnStore) loadAttemptStrokePoints(strokeID int64) ([]learn.StrokePoint, error) {
	rows, err := ls.s.SQL.Query(`
		SELECT x, y FROM attempt_stroke_points WHERE attempt_stroke_id = ? ORDER BY seq ASC
	`, strokeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pts []learn.StrokePoint
	for rows.Next() {
		var p learn.StrokePoint
		if err := rows.Scan(&p.X, &p.Y); err != nil {
			return nil, err
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}

// CountAttemptStrokes returns the number of strokes for an owned attempt (0 if none).
func (ls *LearnStore) CountAttemptStrokes(ctx context.Context, userID, attemptID int64) (int, error) {
	_ = ctx
	if _, err := ls.getAttempt(userID, attemptID); err != nil {
		return 0, err
	}
	var n int
	err := ls.s.SQL.QueryRow(`SELECT COUNT(*) FROM attempt_strokes WHERE attempt_id = ?`, attemptID).Scan(&n)
	return n, err
}

func scanAttempt(row *sql.Row) (learn.Attempt, error) {
	var a learn.Attempt
	var lessonID, clientID sql.NullString
	var cw, ch sql.NullInt64
	var submitted, assessed, abandoned sql.NullTime
	err := row.Scan(
		&a.ID, &a.UserID, &a.CharacterID, &lessonID, &a.Status, &clientID,
		&cw, &ch, &a.StartedAt, &submitted, &assessed, &abandoned, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return learn.Attempt{}, learn.ErrNotFound
		}
		return learn.Attempt{}, err
	}
	if lessonID.Valid {
		a.LessonID = lessonID.String
	}
	if clientID.Valid {
		a.ClientAttemptID = clientID.String
	}
	if cw.Valid {
		v := int(cw.Int64)
		a.CanvasWidth = &v
	}
	if ch.Valid {
		v := int(ch.Int64)
		a.CanvasHeight = &v
	}
	if submitted.Valid {
		t := submitted.Time
		a.SubmittedAt = &t
	}
	if assessed.Valid {
		t := assessed.Time
		a.AssessedAt = &t
	}
	if abandoned.Valid {
		t := abandoned.Time
		a.AbandonedAt = &t
	}
	return a, nil
}

func (ls *LearnStore) SubmitStrokes(ctx context.Context, userID, attemptID int64, strokes []learn.StrokeInput, w, h int) error {
	_ = ctx
	totalPoints := 0
	for _, st := range strokes {
		totalPoints += len(st.Points)
	}
	learnLog("DEBUG", "[learn.AttemptRepo.SubmitStrokes] userID=%d attemptID=%d strokes=%d points=%d", userID, attemptID, len(strokes), totalPoints)

	if err := limits.CheckCanvas(w, h); err != nil {
		return learn.ErrInvalidInput
	}
	if err := limits.ValidateStrokeSet(len(strokes), totalPoints); err != nil {
		return learn.ErrInvalidInput
	}
	for _, st := range strokes {
		color := st.Color
		if color == "" {
			color = "#000000"
		}
		width := st.Width
		if width == 0 {
			width = 2
		}
		if err := limits.ValidateStrokeMeta(width, color, ""); err != nil {
			return learn.ErrInvalidInput
		}
		pts := make([]limits.FloatPoint, len(st.Points))
		for i, p := range st.Points {
			pts[i] = limits.FloatPoint{X: p.X, Y: p.Y}
		}
		if err := limits.ValidateStrokePoints(pts); err != nil {
			return learn.ErrInvalidInput
		}
	}

	tx, err := ls.s.beginImmediate()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var charID string
	err = tx.QueryRow(`SELECT status, character_id FROM practice_attempts WHERE id = ? AND user_id = ?`, attemptID, userID).
		Scan(&status, &charID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return learn.ErrNotFound
		}
		return err
	}
	if status != learn.AttemptStatusDraft {
		learnLog("WARN", "[learn.AttemptRepo.SubmitStrokes] invalid_status attemptID=%d status=%s", attemptID, status)
		return learn.ErrInvalidStatus
	}

	now := time.Now().UTC()
	res, err := tx.Exec(`
		UPDATE practice_attempts
		SET status = ?, canvas_width = ?, canvas_height = ?, submitted_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND status = ?
	`, learn.AttemptStatusSubmitted, w, h, now, now, attemptID, userID, learn.AttemptStatusDraft)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return learn.ErrInvalidStatus
	}

	for seq, st := range strokes {
		color := st.Color
		if color == "" {
			color = "#000000"
		}
		width := st.Width
		if width == 0 {
			width = 2
		}
		r, err := tx.Exec(`
			INSERT INTO attempt_strokes(attempt_id, seq, color, width, started_at_unix_ms)
			VALUES(?, ?, ?, ?, ?)
		`, attemptID, seq, color, width, st.StartedAtUnixMs)
		if err != nil {
			return err
		}
		strokeID, err := r.LastInsertId()
		if err != nil {
			return err
		}
		for pi, p := range st.Points {
			if _, err := tx.Exec(`
				INSERT INTO attempt_stroke_points(attempt_stroke_id, seq, x, y) VALUES(?, ?, ?, ?)
			`, strokeID, pi, p.X, p.Y); err != nil {
				return err
			}
		}
	}

	if err := upsertProgressOnSubmitTx(tx, userID, charID, attemptID); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertProgressOnSubmitTx(tx *sql.Tx, userID int64, characterID string, attemptID int64) error {
	now := time.Now().UTC()
	_, err := tx.Exec(`
		INSERT INTO user_character_progress(user_id, character_id, status, attempt_count, pass_count, last_attempt_id, updated_at)
		VALUES(?, ?, ?, 1, 0, ?, ?)
		ON CONFLICT(user_id, character_id) DO UPDATE SET
			status = CASE
				WHEN user_character_progress.status = 'passed' THEN 'passed'
				ELSE 'practicing'
			END,
			attempt_count = user_character_progress.attempt_count + 1,
			last_attempt_id = excluded.last_attempt_id,
			updated_at = excluded.updated_at
	`, userID, characterID, learn.ProgressStatusPracticing, attemptID, now)
	return err
}

func (ls *LearnStore) MarkAssessed(ctx context.Context, userID, attemptID int64) error {
	_ = ctx
	tx, err := ls.s.beginImmediate()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	res, err := tx.Exec(`
		UPDATE practice_attempts
		SET status = ?, assessed_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND status = ?
	`, learn.AttemptStatusAssessed, now, now, attemptID, userID, learn.AttemptStatusSubmitted)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var status string
		err := tx.QueryRow(`SELECT status FROM practice_attempts WHERE id = ? AND user_id = ?`, attemptID, userID).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return learn.ErrNotFound
		}
		if err != nil {
			return err
		}
		learnLog("WARN", "[learn.AttemptRepo.MarkAssessed] invalid_status attemptID=%d status=%s", attemptID, status)
		return learn.ErrInvalidStatus
	}
	return tx.Commit()
}

func (ls *LearnStore) Abandon(ctx context.Context, userID, attemptID int64) error {
	_ = ctx
	tx, err := ls.s.beginImmediate()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	res, err := tx.Exec(`
		UPDATE practice_attempts
		SET status = ?, abandoned_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND status = ?
	`, learn.AttemptStatusAbandoned, now, now, attemptID, userID, learn.AttemptStatusDraft)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var status string
		err := tx.QueryRow(`SELECT status FROM practice_attempts WHERE id = ? AND user_id = ?`, attemptID, userID).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return learn.ErrNotFound
		}
		if err != nil {
			return err
		}
		return learn.ErrInvalidStatus
	}
	return tx.Commit()
}

func (ls *LearnStore) SaveResult(ctx context.Context, userID int64, in learn.SaveAssessment) (learn.AssessmentResult, error) {
	_ = ctx
	learnLog("DEBUG", "[learn.AssessmentRepo.SaveResult] userID=%d attemptID=%d", userID, in.AttemptID)

	if in.ScoreKind == "" || in.Assessor == "" || in.SetID == "" {
		return learn.AssessmentResult{}, learn.ErrInvalidInput
	}
	if len(in.Feedback) > 2 {
		return learn.AssessmentResult{}, learn.ErrInvalidInput
	}

	tx, err := ls.s.beginImmediate()
	if err != nil {
		return learn.AssessmentResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var status, characterID string
	err = tx.QueryRow(`SELECT status, character_id FROM practice_attempts WHERE id = ? AND user_id = ?`, in.AttemptID, userID).
		Scan(&status, &characterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return learn.AssessmentResult{}, learn.ErrNotFound
		}
		return learn.AssessmentResult{}, err
	}
	if status != learn.AttemptStatusSubmitted {
		if status == learn.AttemptStatusAssessed {
			learnLog("WARN", "[learn.AssessmentRepo.SaveResult] conflict already assessed attemptID=%d", in.AttemptID)
			return learn.AssessmentResult{}, learn.ErrConflict
		}
		learnLog("WARN", "[learn.AssessmentRepo.SaveResult] invalid_status attemptID=%d status=%s", in.AttemptID, status)
		return learn.AssessmentResult{}, learn.ErrInvalidStatus
	}

	reasonsJSON, err := json.Marshal(in.Reasons)
	if err != nil {
		return learn.AssessmentResult{}, err
	}
	passInt := 0
	if in.Pass {
		passInt = 1
	}

	now := time.Now().UTC()
	res, err := tx.Exec(`
		INSERT INTO assessment_results(attempt_id, pass, score, score_kind, assessor, set_id, reasons_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
	`, in.AttemptID, passInt, in.Score, in.ScoreKind, in.Assessor, in.SetID, string(reasonsJSON), now)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return learn.AssessmentResult{}, learn.ErrConflict
		}
		return learn.AssessmentResult{}, err
	}
	assessID, err := res.LastInsertId()
	if err != nil {
		return learn.AssessmentResult{}, err
	}

	for _, fb := range in.Feedback {
		if _, err := tx.Exec(`
			INSERT INTO assessment_feedback(assessment_id, rank, code, message) VALUES(?, ?, ?, ?)
		`, assessID, fb.Rank, fb.Code, fb.Message); err != nil {
			return learn.AssessmentResult{}, err
		}
	}

	_, err = tx.Exec(`
		UPDATE practice_attempts
		SET status = ?, assessed_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, learn.AttemptStatusAssessed, now, now, in.AttemptID, userID)
	if err != nil {
		return learn.AssessmentResult{}, err
	}

	if err := upsertProgressOnAssessTx(tx, userID, characterID, in.AttemptID, in.Pass, now); err != nil {
		return learn.AssessmentResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return learn.AssessmentResult{}, err
	}

	learnLog("INFO", "[learn.AssessmentRepo.SaveResult] userID=%d attemptID=%d pass=%t score=%.3f", userID, in.AttemptID, in.Pass, in.Score)

	out, err := ls.GetByAttempt(ctx, userID, in.AttemptID)
	if err != nil {
		return learn.AssessmentResult{}, err
	}
	return *out, nil
}

func upsertProgressOnAssessTx(tx *sql.Tx, userID int64, characterID string, attemptID int64, pass bool, now time.Time) error {
	passInc := 0
	if pass {
		passInc = 1
	}
	var lastPassed any
	if pass {
		lastPassed = now
	}
	status := learn.ProgressStatusPracticing
	if pass {
		status = learn.ProgressStatusPassed
	}
	_, err := tx.Exec(`
		INSERT INTO user_character_progress(user_id, character_id, status, attempt_count, pass_count, last_attempt_id, last_passed_at, updated_at)
		VALUES(?, ?, ?, 0, ?, ?, ?, ?)
		ON CONFLICT(user_id, character_id) DO UPDATE SET
			status = CASE
				WHEN excluded.pass_count > 0 OR user_character_progress.pass_count > 0 THEN 'passed'
				ELSE 'practicing'
			END,
			pass_count = user_character_progress.pass_count + excluded.pass_count,
			last_attempt_id = excluded.last_attempt_id,
			last_passed_at = COALESCE(excluded.last_passed_at, user_character_progress.last_passed_at),
			updated_at = excluded.updated_at
	`, userID, characterID, status, passInc, attemptID, lastPassed, now)
	_ = status // status used in INSERT; CASE handles conflict
	return err
}

func (ls *LearnStore) GetByAttempt(ctx context.Context, userID, attemptID int64) (*learn.AssessmentResult, error) {
	_ = ctx
	// Ownership: attempt must belong to user.
	var dummy int
	err := ls.s.SQL.QueryRow(`SELECT 1 FROM practice_attempts WHERE id = ? AND user_id = ?`, attemptID, userID).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, learn.ErrNotFound
		}
		return nil, err
	}

	row := ls.s.SQL.QueryRow(`
		SELECT id, attempt_id, pass, score, score_kind, assessor, set_id, reasons_json, created_at
		FROM assessment_results WHERE attempt_id = ?
	`, attemptID)
	var ar learn.AssessmentResult
	var passInt int
	var reasonsRaw string
	if err := row.Scan(&ar.ID, &ar.AttemptID, &passInt, &ar.Score, &ar.ScoreKind, &ar.Assessor, &ar.SetID, &reasonsRaw, &ar.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, learn.ErrNotFound
		}
		return nil, err
	}
	ar.Pass = passInt != 0
	if reasonsRaw != "" {
		if err := json.Unmarshal([]byte(reasonsRaw), &ar.Reasons); err != nil {
			return nil, fmt.Errorf("reasons_json: %w", err)
		}
	}

	frows, err := ls.s.SQL.Query(`
		SELECT rank, code, message FROM assessment_feedback WHERE assessment_id = ? ORDER BY rank
	`, ar.ID)
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	for frows.Next() {
		var fb learn.FeedbackItem
		if err := frows.Scan(&fb.Rank, &fb.Code, &fb.Message); err != nil {
			return nil, err
		}
		ar.Feedback = append(ar.Feedback, fb)
	}
	return &ar, frows.Err()
}

func (ls *LearnStore) getProgress(ctx context.Context, userID int64, characterID string) (*learn.Progress, error) {
	_ = ctx
	row := ls.s.SQL.QueryRow(`
		SELECT user_id, character_id, status, attempt_count, pass_count, last_attempt_id, last_passed_at, updated_at
		FROM user_character_progress WHERE user_id = ? AND character_id = ?
	`, userID, characterID)
	return scanProgress(row)
}

func scanProgress(row *sql.Row) (*learn.Progress, error) {
	var p learn.Progress
	var lastAttempt sql.NullInt64
	var lastPassed sql.NullTime
	err := row.Scan(&p.UserID, &p.CharacterID, &p.Status, &p.AttemptCount, &p.PassCount, &lastAttempt, &lastPassed, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, learn.ErrNotFound
		}
		return nil, err
	}
	if lastAttempt.Valid {
		v := lastAttempt.Int64
		p.LastAttemptID = &v
	}
	if lastPassed.Valid {
		t := lastPassed.Time
		p.LastPassedAt = &t
	}
	return &p, nil
}

// ProgressGet implements ProgressRepo.Get without colliding with CharacterRepo.Get / AttemptRepo.Get.
// Use typed wrappers below for interface satisfaction.

type characterRepo struct{ *LearnStore }
type lessonRepo struct{ *LearnStore }
type attemptRepo struct{ *LearnStore }
type assessmentRepo struct{ *LearnStore }
type progressRepo struct{ *LearnStore }

func (r characterRepo) Get(ctx context.Context, id string) (*learn.Character, error) {
	return r.LearnStore.getCharacter(ctx, id)
}
func (r characterRepo) ListBySet(ctx context.Context, setID string) ([]learn.Character, error) {
	return r.LearnStore.ListBySet(ctx, setID)
}
func (r lessonRepo) GetPublished(ctx context.Context, id string) (*learn.Lesson, error) {
	return r.LearnStore.GetPublished(ctx, id)
}
func (r lessonRepo) ListCharacters(ctx context.Context, lessonID string) ([]learn.LessonCharacter, error) {
	return r.LearnStore.ListCharacters(ctx, lessonID)
}
func (r attemptRepo) CreateDraft(ctx context.Context, in learn.CreateDraft) (learn.Attempt, error) {
	return r.LearnStore.CreateDraft(ctx, in)
}
func (r attemptRepo) Get(ctx context.Context, userID, attemptID int64) (*learn.Attempt, error) {
	a, err := r.LearnStore.getAttempt(userID, attemptID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
func (r attemptRepo) GetByClientAttemptID(ctx context.Context, userID int64, clientAttemptID string) (*learn.Attempt, error) {
	return r.LearnStore.GetByClientAttemptID(ctx, userID, clientAttemptID)
}
func (r attemptRepo) SubmitStrokes(ctx context.Context, userID, attemptID int64, strokes []learn.StrokeInput, w, h int) error {
	return r.LearnStore.SubmitStrokes(ctx, userID, attemptID, strokes, w, h)
}
func (r attemptRepo) ListStrokes(ctx context.Context, userID, attemptID int64) ([]learn.StrokeInput, int, int, error) {
	return r.LearnStore.ListStrokes(ctx, userID, attemptID)
}
func (r attemptRepo) MarkAssessed(ctx context.Context, userID, attemptID int64) error {
	return r.LearnStore.MarkAssessed(ctx, userID, attemptID)
}
func (r attemptRepo) Abandon(ctx context.Context, userID, attemptID int64) error {
	return r.LearnStore.Abandon(ctx, userID, attemptID)
}
func (r assessmentRepo) SaveResult(ctx context.Context, userID int64, in learn.SaveAssessment) (learn.AssessmentResult, error) {
	return r.LearnStore.SaveResult(ctx, userID, in)
}
func (r assessmentRepo) GetByAttempt(ctx context.Context, userID, attemptID int64) (*learn.AssessmentResult, error) {
	return r.LearnStore.GetByAttempt(ctx, userID, attemptID)
}
func (r progressRepo) Get(ctx context.Context, userID int64, characterID string) (*learn.Progress, error) {
	return r.LearnStore.getProgress(ctx, userID, characterID)
}
func (r progressRepo) ListForUser(ctx context.Context, userID int64) ([]learn.Progress, error) {
	_ = ctx
	rows, err := r.s.SQL.Query(`
		SELECT user_id, character_id, status, attempt_count, pass_count, last_attempt_id, last_passed_at, updated_at
		FROM user_character_progress WHERE user_id = ? ORDER BY character_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []learn.Progress
	for rows.Next() {
		var p learn.Progress
		var lastAttempt sql.NullInt64
		var lastPassed sql.NullTime
		if err := rows.Scan(&p.UserID, &p.CharacterID, &p.Status, &p.AttemptCount, &p.PassCount, &lastAttempt, &lastPassed, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if lastAttempt.Valid {
			v := lastAttempt.Int64
			p.LastAttemptID = &v
		}
		if lastPassed.Valid {
			t := lastPassed.Time
			p.LastPassedAt = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Characters returns the CharacterRepo view.
func (ls *LearnStore) Characters() learn.CharacterRepo { return characterRepo{ls} }

// Lessons returns the LessonRepo view.
func (ls *LearnStore) Lessons() learn.LessonRepo { return lessonRepo{ls} }

// Attempts returns the AttemptRepo view.
func (ls *LearnStore) Attempts() learn.AttemptRepo { return attemptRepo{ls} }

// Assessments returns the AssessmentRepo view.
func (ls *LearnStore) Assessments() learn.AssessmentRepo { return assessmentRepo{ls} }

// Progress returns the ProgressRepo view.
func (ls *LearnStore) Progress() learn.ProgressRepo { return progressRepo{ls} }
