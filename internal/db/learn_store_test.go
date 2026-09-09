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

func TestCreateDraftIdempotentClientAttemptID(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "idem-create.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("idem@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, err := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	if err != nil || len(chars) < 2 {
		t.Fatalf("chars: %v", err)
	}

	first, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID: uid, CharacterID: chars[0].ID, ClientAttemptID: "client-attempt-1",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	replay, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID: uid, CharacterID: chars[0].ID, ClientAttemptID: "client-attempt-1",
	})
	if err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	if replay.ID != first.ID {
		t.Fatalf("replay id=%d want %d", replay.ID, first.ID)
	}

	_, err = ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID: uid, CharacterID: chars[1].ID, ClientAttemptID: "client-attempt-1",
	})
	if !errors.Is(err, learn.ErrConflict) {
		t.Fatalf("mismatch want conflict, got %v", err)
	}
}

func TestListStrokesAfterSubmit(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "list-strokes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("list@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateUser("other-list@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	draft, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[0].ID})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ls.Attempts().ListStrokes(ctx, uid, draft.ID)
	if !errors.Is(err, learn.ErrInvalidStatus) {
		t.Fatalf("draft list want invalid_status, got %v", err)
	}

	in := []learn.StrokeInput{
		{Color: "#111111", Width: 3, Points: []learn.StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}}},
		{Color: "#222222", Width: 4, Points: []learn.StrokePoint{{X: 5, Y: 6}}},
	}
	if err := ls.Attempts().SubmitStrokes(ctx, uid, draft.ID, in, 320, 240); err != nil {
		t.Fatalf("submit: %v", err)
	}
	got, w, h, err := ls.Attempts().ListStrokes(ctx, uid, draft.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if w != 320 || h != 240 || len(got) != 2 {
		t.Fatalf("w=%d h=%d strokes=%d", w, h, len(got))
	}
	if got[0].Color != "#111111" || len(got[0].Points) != 2 || got[1].Points[0].X != 5 {
		t.Fatalf("stroke payload: %+v", got)
	}

	_, _, _, err = ls.Attempts().ListStrokes(ctx, other, draft.ID)
	if !errors.Is(err, learn.ErrNotFound) {
		t.Fatalf("other user want not_found, got %v", err)
	}
}


func TestListAttemptsPaginationAndFilters(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "list-hist.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, _ := store.CreateUser("hist@example.com", "hash")
	other, _ := store.CreateUser("hist-other@example.com", "hash")
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	lessonID := "lesson:hiragana5"

	makeAssessed := func(charID string, pass bool) int64 {
		d, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: charID, LessonID: lessonID})
		if err != nil {
			t.Fatalf("draft: %v", err)
		}
		if err := ls.Attempts().SubmitStrokes(ctx, uid, d.ID, []learn.StrokeInput{
			{Width: 2, Color: "#000000", Points: []learn.StrokePoint{{X: 1, Y: 1}}},
		}, 300, 300); err != nil {
			t.Fatalf("submit: %v", err)
		}
		_, err = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
			AttemptID: d.ID, Pass: pass, Score: 0.5, ScoreKind: recognize.ScoreKindMatch,
			Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
			Feedback: []learn.FeedbackItem{{Rank: 1, Code: "stroke_count", Message: "check count"}},
		})
		if err != nil {
			t.Fatalf("assess: %v", err)
		}
		return d.ID
	}

	id1 := makeAssessed(chars[0].ID, true)
	id2 := makeAssessed(chars[0].ID, false)
	_ = makeAssessed(chars[1].ID, true)

	abandoned, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[0].ID, LessonID: lessonID})
	_ = ls.Attempts().Abandon(ctx, uid, abandoned.ID)

	page, err := ls.Attempts().List(ctx, uid, learn.AttemptListFilter{Limit: 1})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || !page.HasNext || page.NextID == nil {
		t.Fatalf("page1: %+v", page)
	}
	if page.Items[0].Pass == nil {
		t.Fatal("assessed row missing pass")
	}
	// Ensure no points field exists on history item (struct has none) — glyph present.
	if page.Items[0].Glyph == "" {
		t.Fatal("missing glyph")
	}

	page2, err := ls.Attempts().List(ctx, uid, learn.AttemptListFilter{
		Limit: 1, AfterStartedAt: page.NextStartedAt, AfterID: page.NextID,
	})
	if err != nil || len(page2.Items) != 1 {
		t.Fatalf("page2: %+v err=%v", page2, err)
	}
	if page2.Items[0].ID == page.Items[0].ID {
		t.Fatal("cursor did not advance")
	}

	filtered, err := ls.Attempts().List(ctx, uid, learn.AttemptListFilter{
		CharacterID: chars[0].ID, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range filtered.Items {
		if it.CharacterID != chars[0].ID {
			t.Fatalf("filter leak: %s", it.CharacterID)
		}
		if it.Status != learn.AttemptStatusAssessed {
			t.Fatalf("default status filter: %s", it.Status)
		}
	}
	if len(filtered.Items) != 2 {
		t.Fatalf("char0 assessed count=%d want 2 (ids %d %d)", len(filtered.Items), id1, id2)
	}

	withAbandon, err := ls.Attempts().List(ctx, uid, learn.AttemptListFilter{
		Statuses: []string{learn.AttemptStatusAssessed, learn.AttemptStatusAbandoned},
		Limit:    50,
	})
	if err != nil {
		t.Fatal(err)
	}
	foundAbandon := false
	for _, it := range withAbandon.Items {
		if it.Status == learn.AttemptStatusAbandoned {
			foundAbandon = true
			if it.Pass != nil || it.Score != nil {
				t.Fatalf("abandoned should omit assessment: %+v", it)
			}
		}
	}
	if !foundAbandon {
		t.Fatal("expected abandoned in optional filter")
	}

	cross, err := ls.Attempts().List(ctx, other, learn.AttemptListFilter{Limit: 20})
	if err != nil || len(cross.Items) != 0 {
		t.Fatalf("cross-user list: %+v err=%v", cross, err)
	}
}

func TestClearPracticeDataKeepsBoard(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "clear-practice.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, _ := store.CreateUser("clearprac@example.com", "hash")
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	d, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: chars[0].ID})
	_ = ls.Attempts().SubmitStrokes(ctx, uid, d.ID, []learn.StrokeInput{
		{Width: 2, Color: "#000000", Points: []learn.StrokePoint{{X: 1, Y: 1}}},
	}, 300, 300)
	_, _ = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
		AttemptID: d.ID, Pass: true, Score: 0.9, ScoreKind: recognize.ScoreKindMatch,
		Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
	})
	_, err = store.ApplyStrokeCreate(uid, 0, "board-keep-1", "#000000", 2, 1, []StrokePoint{{X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("board stroke: %v", err)
	}

	res, err := ls.Attempts().ClearPracticeData(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}
	if res.AttemptsDeleted < 1 || res.ProgressRowsCleared < 1 {
		t.Fatalf("clear result: %+v", res)
	}
	list, _ := ls.Attempts().List(ctx, uid, learn.AttemptListFilter{Limit: 20})
	if len(list.Items) != 0 {
		t.Fatalf("history remain: %d", len(list.Items))
	}
	_, err = ls.Progress().Get(ctx, uid, chars[0].ID)
	if !errors.Is(err, learn.ErrNotFound) {
		t.Fatalf("progress after clear: %v", err)
	}
	board, _ := store.ListStrokesByUser(uid)
	if len(board) != 1 {
		t.Fatalf("board strokes=%d want 1", len(board))
	}
}

func TestListAssessedOutcomesForMastery(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "outcomes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.SQL.Close()
	uid, _ := store.CreateUser("out@example.com", "hash")
	ls := NewLearnStore(store)
	ctx := context.Background()
	chars, _ := ls.Characters().ListBySet(ctx, recognize.SetIDHiragana5)
	cid := chars[0].ID

	for _, pass := range []bool{false, true, true} {
		d, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: cid})
		_ = ls.Attempts().SubmitStrokes(ctx, uid, d.ID, []learn.StrokeInput{
			{Width: 2, Color: "#000000", Points: []learn.StrokePoint{{X: 1, Y: 1}}},
		}, 300, 300)
		_, _ = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
			AttemptID: d.ID, Pass: pass, Score: 0.5, ScoreKind: recognize.ScoreKindMatch,
			Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
		})
	}
	abandoned, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: cid})
	_ = ls.Attempts().Abandon(ctx, uid, abandoned.ID)

	m, err := ls.Progress().ListAssessedOutcomes(ctx, uid, []string{cid}, 20)
	if err != nil {
		t.Fatal(err)
	}
	got := learn.DeriveMastery(m[cid])
	if got.State != learn.MasteryStateSteady || got.AssessedCount != 3 {
		t.Fatalf("mastery=%+v outcomes=%+v", got, m[cid])
	}
}
