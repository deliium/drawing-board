package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/deliium/drawing-board/internal/learn"
	"github.com/gorilla/mux"
)

func TestGetLesson_Hiragana5Published(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, _, uid := newAttemptTestAPI(t, "lesson-get")

	req := attemptSessionReq(t, api, http.MethodGet, "/api/lessons/lesson:hiragana5", uid, "")
	req = mux.SetURLVars(req, map[string]string{"id": "lesson:hiragana5"})
	rec := httptest.NewRecorder()
	api.GetLesson(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp lessonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "lesson:hiragana5" || resp.SetID != "hiragana5" {
		t.Fatalf("lesson meta: %+v", resp)
	}
	if resp.ContentVersion == "" {
		t.Fatal("expected contentVersion")
	}
	if len(resp.Characters) != 5 {
		t.Fatalf("characterCount=%d want 5", len(resp.Characters))
	}

	want := map[string]struct {
		glyph       string
		strokeCount int
		exampleWord string
	}{
		"hira:あ": {glyph: "あ", strokeCount: 3, exampleWord: "あさ"},
		"hira:い": {glyph: "い", strokeCount: 2, exampleWord: "いぬ"},
		"hira:う": {glyph: "う", strokeCount: 2, exampleWord: "うみ"},
		"hira:え": {glyph: "え", strokeCount: 2, exampleWord: "えき"},
		"hira:お": {glyph: "お", strokeCount: 3, exampleWord: "おと"},
	}
	for i, ch := range resp.Characters {
		exp, ok := want[ch.ID]
		if !ok {
			t.Fatalf("unexpected character %s", ch.ID)
		}
		if ch.Glyph != exp.glyph || ch.StrokeCount != exp.strokeCount || ch.Example.Word != exp.exampleWord {
			t.Fatalf("char[%d]=%+v want glyph=%s strokes=%d word=%s", i, ch, exp.glyph, exp.strokeCount, exp.exampleWord)
		}
		if ch.Position < 1 {
			t.Fatalf("position=%d for %s", ch.Position, ch.ID)
		}
		var pron map[string]any
		if err := json.Unmarshal(ch.Pronunciation, &pron); err != nil {
			t.Fatalf("pronunciation for %s: %v raw=%s", ch.ID, err, string(ch.Pronunciation))
		}
		if _, ok := pron["ipa"]; !ok {
			t.Fatalf("missing ipa in pronunciation for %s: %v", ch.ID, pron)
		}
		audioRef, _ := pron["audioRef"].(string)
		if audioRef == "" {
			t.Fatalf("missing audioRef for %s: %v", ch.ID, pron)
		}
		if ch.GuidanceEn == "" || ch.GuidanceJa == "" {
			t.Fatalf("missing guidance for %s en=%q ja=%q", ch.ID, ch.GuidanceEn, ch.GuidanceJa)
		}
	}
	if resp.ContentVersion != "hiragana5-content-v2" {
		t.Fatalf("contentVersion=%q", resp.ContentVersion)
	}
}

func TestGetLesson_NotFoundAndUnauthorized(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "lesson-404")

	req := attemptSessionReq(t, api, http.MethodGet, "/api/lessons/lesson:missing", uid, "")
	req = mux.SetURLVars(req, map[string]string{"id": "lesson:missing"})
	rec := httptest.NewRecorder()
	api.GetLesson(rec, req)
	if rec.Code != 404 {
		t.Fatalf("missing lesson code=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/lessons/lesson:hiragana5", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "lesson:hiragana5"})
	rec = httptest.NewRecorder()
	api.GetLesson(rec, req)
	if rec.Code != 401 {
		t.Fatalf("unauth code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListProgress_EmptyThenAfterPass(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, _, uid := newAttemptTestAPI(t, "progress-list")

	req := attemptSessionReq(t, api, http.MethodGet, "/api/progress?lessonId=lesson:hiragana5", uid, "")
	rec := httptest.NewRecorder()
	api.ListProgress(rec, req)
	if rec.Code != 200 {
		t.Fatalf("empty list code=%d body=%s", rec.Code, rec.Body.String())
	}
	var empty progressListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &empty)
	if len(empty.Items) != 0 {
		t.Fatalf("expected empty progress, got %+v", empty.Items)
	}

	ls := api.learnStore()
	ctx := context.Background()
	at, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
		UserID:      uid,
		CharacterID: "hira:あ",
		LessonID:    "lesson:hiragana5",
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := ls.Attempts().SubmitStrokes(ctx, uid, at.ID, []learn.StrokeInput{
		{Color: "#000", Width: 2, Points: []learn.StrokePoint{{X: 10, Y: 10}, {X: 20, Y: 20}}},
	}, 300, 300); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
		AttemptID: at.ID,
		Pass:      true,
		Score:     0.9,
		ScoreKind: "match",
		Assessor:  learn.AssessorTargetCompare,
		SetID:     "hiragana5",
		Reasons:   []string{"ok"},
		Feedback:  nil,
	}); err != nil {
		t.Fatalf("save result: %v", err)
	}

	req = attemptSessionReq(t, api, http.MethodGet, "/api/progress?setId=hiragana5", uid, "")
	rec = httptest.NewRecorder()
	api.ListProgress(rec, req)
	if rec.Code != 200 {
		t.Fatalf("after pass code=%d body=%s", rec.Code, rec.Body.String())
	}
	var after progressListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &after)
	if len(after.Items) != 1 {
		t.Fatalf("progress items=%d body=%s", len(after.Items), rec.Body.String())
	}
	item := after.Items[0]
	if item.CharacterID != "hira:あ" || item.Status != learn.ProgressStatusPassed || item.PassCount < 1 {
		t.Fatalf("unexpected progress item: %+v", item)
	}
	if item.LastAttemptID == nil || *item.LastAttemptID != at.ID {
		t.Fatalf("lastAttemptId=%v want %s", item.LastAttemptID, at.ID)
	}
	if item.Mastery == nil || item.Mastery.State != learn.MasteryStatePassedOnce {
		t.Fatalf("mastery=%+v want passed_once", item.Mastery)
	}
	if item.Review == nil || item.Review.Box != 1 || item.Review.DueAt == nil || item.Review.IntervalDays != 1 {
		t.Fatalf("review=%+v want box=1 intervalDays=1", item.Review)
	}

	req = attemptSessionReq(t, api, http.MethodGet, "/api/progress?lessonId=lesson:missing", uid, "")
	rec = httptest.NewRecorder()
	api.ListProgress(rec, req)
	if rec.Code != 404 {
		t.Fatalf("bad lesson filter code=%d", rec.Code)
	}
}

func TestGetLesson_UsesSeededDB(t *testing.T) {
	// Sanity: Open seeds hiragana5 so lesson GETs work without extra setup.
	api, store, _ := newTestAPI(t, filepath.Join(t.TempDir(), "lesson-seed.db"))
	api.Learn = nil // force learnStore() path
	_ = store
	ls := api.learnStore()
	lesson, err := ls.Lessons().GetPublished(context.Background(), "lesson:hiragana5")
	if err != nil {
		t.Fatalf("seeded lesson: %v", err)
	}
	if lesson.Title == "" {
		t.Fatal("empty lesson title")
	}
}
