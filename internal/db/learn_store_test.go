package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/deliium/drawing-board/internal/learn"
	"github.com/deliium/drawing-board/internal/recognize"
)

func TestLearnStoreAttemptLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "learn.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.SQL.Close()

	uid, err := store.CreateUser("learner@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	ls := NewLearnStore(store)
	ctx := context.Background()

	chars, err := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	if err != nil || len(chars) != 5 {
		t.Fatalf("ListBySet: len=%d err=%v", len(chars), err)
	}
	if chars[0].Romanization == "" || chars[0].DescriptionEn == "" || chars[0].ContentVersion == "" {
		t.Fatalf("pedagogy fields empty: %+v", chars[0])
	}
	lesson, err := ls.Lessons().GetPublished(ctx, "lesson:hiragana5")
	if err != nil {
		t.Fatalf("GetPublished: %v", err)
	}
	ordered, err := ls.Lessons().ListCharacters(ctx, lesson.ID)
	if err != nil || len(ordered) != 5 {
		t.Fatalf("ListCharacters: len=%d err=%v", len(ordered), err)
	}
	for i, lc := range ordered {
		if lc.Position != i+1 {
			t.Fatalf("position[%d]=%d want %d", i, lc.Position, i+1)
		}
		if lc.Glyph != chars[i].Glyph {
			t.Fatalf("order mismatch glyph %q vs %q", lc.Glyph, chars[i].Glyph)
		}
	}

	charID := chars[0].ID
	draft, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID:      uid,
		CharacterID: charID,
		LessonID:    lesson.ID,
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if draft.Status != learn.AttemptStatusDraft {
		t.Fatalf("status=%s", draft.Status)
	}

	strokes := []learn.StrokeInput{
		{Color: "#000000", Width: 2, Points: []learn.StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}}},
		{Color: "#000000", Width: 2, Points: []learn.StrokePoint{{X: 5, Y: 6}}},
	}
	if err := ls.Attempts().SubmitStrokes(ctx, uid, draft.ID, strokes, 300, 300); err != nil {
		t.Fatalf("SubmitStrokes: %v", err)
	}
	got, err := ls.Attempts().Get(ctx, uid, draft.ID)
	if err != nil || got.Status != learn.AttemptStatusSubmitted {
		t.Fatalf("after submit status=%v err=%v", got, err)
	}

	ar, err := ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
		AttemptID: draft.ID,
		Pass:      true,
		Score:     0.91,
		ScoreKind: recognize.ScoreKindMatch,
		Assessor:  learn.AssessorTargetCompare,
		SetID:     recognize.SetIDHiragana5,
		Reasons:   []string{"top_match"},
		Feedback:  []learn.FeedbackItem{{Rank: 1, Code: "stroke_count", Message: ""}},
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}
	if !ar.Pass || ar.ScoreKind != recognize.ScoreKindMatch {
		t.Fatalf("assessment: %+v", ar)
	}

	prog, err := ls.Progress().Get(ctx, uid, charID)
	if err != nil {
		t.Fatalf("Progress.Get: %v", err)
	}
	if prog.Status != learn.ProgressStatusPassed || prog.PassCount < 1 || prog.AttemptCount < 1 {
		t.Fatalf("progress: %+v", prog)
	}

	_, err = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
		AttemptID: draft.ID,
		Pass:      false,
		Score:     0.1,
		ScoreKind: recognize.ScoreKindMatch,
		Assessor:  learn.AssessorTargetCompare,
		SetID:     recognize.SetIDHiragana5,
		Reasons:   nil,
	})
	if !errors.Is(err, learn.ErrConflict) {
		t.Fatalf("second assess want conflict, got %v", err)
	}
}

func TestBoardClearLeavesAttempts(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "clear-iso.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("cleariso@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	draft, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := ls.Attempts().SubmitStrokes(ctx, uid, draft.ID, []learn.StrokeInput{
		{Width: 2, Color: "#000000", Points: []learn.StrokePoint{{X: 0, Y: 0}}},
	}, 300, 300); err != nil {
		t.Fatal(err)
	}

	_, err = store.ApplyStrokeCreate(uid, 0, "board-op-1", "#000000", 2, 1, []StrokePoint{{X: 9, Y: 9}})
	if err != nil {
		t.Fatalf("board create: %v", err)
	}
	rev, _ := store.GetBoardRev(uid)
	if _, err := store.ApplyClear(uid, rev, "board-clear-1"); err != nil {
		t.Fatalf("ApplyClear: %v", err)
	}
	boardStrokes, _ := store.ListStrokesByUser(uid)
	if len(boardStrokes) != 0 {
		t.Fatalf("board strokes after clear: %d", len(boardStrokes))
	}

	got, err := ls.Attempts().Get(ctx, uid, draft.ID)
	if err != nil {
		t.Fatalf("attempt missing after clear: %v", err)
	}
	if got.Status != learn.AttemptStatusSubmitted {
		t.Fatalf("attempt status=%s", got.Status)
	}
	var strokeCount int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM attempt_strokes WHERE attempt_id=?`, draft.ID).Scan(&strokeCount)
	if strokeCount != 1 {
		t.Fatalf("attempt strokes=%d", strokeCount)
	}
}

func TestUserDeleteCascadesLearning(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "cascade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("cascade@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	draft, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[0].ID})
	_ = ls.Attempts().SubmitStrokes(ctx, uid, draft.ID, []learn.StrokeInput{
		{Width: 2, Color: "#000000", Points: []learn.StrokePoint{{X: 1, Y: 1}}},
	}, 300, 300)
	_, _ = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
		AttemptID: draft.ID, Pass: true, Score: 0.8,
		ScoreKind: recognize.ScoreKindMatch, Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
	})

	if _, err := store.SQL.Exec(`DELETE FROM users WHERE id=?`, uid); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var n int
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM practice_attempts WHERE user_id=?`, uid).Scan(&n)
	if n != 0 {
		t.Fatalf("attempts remain: %d", n)
	}
	_ = store.SQL.QueryRow(`SELECT COUNT(1) FROM user_character_progress WHERE user_id=?`, uid).Scan(&n)
	if n != 0 {
		t.Fatalf("progress remain: %d", n)
	}
}

func TestCharacterDeleteRestrictedByAttempt(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "restrict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, _ := store.CreateUser("restrict@example.com", "hash")
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	_, _ = ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[0].ID})

	_, err = store.SQL.Exec(`DELETE FROM characters WHERE id=?`, chars[0].ID)
	if err == nil {
		t.Fatal("expected RESTRICT on character delete")
	}
}

func TestAttemptOwnershipIsolation(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "own.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	a, _ := store.CreateUser("a@example.com", "hash")
	b, _ := store.CreateUser("b@example.com", "hash")
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	draft, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: a, CharacterID: chars[0].ID})

	_, err = ls.Attempts().Get(ctx, b, draft.ID)
	if !errors.Is(err, learn.ErrNotFound) {
		t.Fatalf("cross-user get: %v", err)
	}
}
