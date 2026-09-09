package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/learn"
	"github.com/deliium/drawing-board/internal/recognize"
)

func TestListAttempts_HistoryNoPoints(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, _, uid := newAttemptTestAPI(t, "hist-list")
	ls := api.learnStore()
	ctx := context.Background()

	for i, pass := range []bool{true, false} {
		d, err := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{
			UserID: uid, CharacterID: "hira:あ", LessonID: "lesson:hiragana5",
			ClientAttemptID: "hist-" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := ls.Attempts().SubmitStrokes(ctx, uid, d.ID, []learn.StrokeInput{
			{Width: 2, Color: "#000", Points: []learn.StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}}},
		}, 300, 300); err != nil {
			t.Fatal(err)
		}
		_, err = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
			AttemptID: d.ID, Pass: pass, Score: 0.55, ScoreKind: recognize.ScoreKindMatch,
			Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
			Feedback: []learn.FeedbackItem{{Rank: 1, Code: "stroke_count", Message: "count"}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	req := attemptSessionReq(t, api, http.MethodGet, "/api/attempts?limit=1", uid, "")
	rec := httptest.NewRecorder()
	api.ListAttempts(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if strings.Contains(raw, `"points"`) {
		t.Fatalf("list payload must not include points: %s", raw)
	}
	var page1 attemptHistoryListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &page1); err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 1 || page1.NextCursor == nil {
		t.Fatalf("page1=%+v", page1)
	}

	req = attemptSessionReq(t, api, http.MethodGet, "/api/attempts?limit=1&cursor="+*page1.NextCursor, uid, "")
	rec = httptest.NewRecorder()
	api.ListAttempts(rec, req)
	var page2 attemptHistoryListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &page2)
	if len(page2.Items) != 1 || page2.Items[0].ID == page1.Items[0].ID {
		t.Fatalf("page2=%+v page1=%+v", page2, page1)
	}

	otherAPI, _, otherUID := newAttemptTestAPI(t, "hist-other")
	req = attemptSessionReq(t, otherAPI, http.MethodGet, "/api/attempts", otherUID, "")
	rec = httptest.NewRecorder()
	otherAPI.ListAttempts(rec, req)
	var empty attemptHistoryListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &empty)
	if len(empty.Items) != 0 {
		t.Fatalf("foreign list leaked: %+v", empty.Items)
	}
}

func TestProgressNextAndClear(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, store, uid := newAttemptTestAPI(t, "next-clear")
	ls := api.learnStore()
	ctx := context.Background()

	req := attemptSessionReq(t, api, http.MethodGet, "/api/progress/next?lessonId=lesson:hiragana5", uid, "")
	rec := httptest.NewRecorder()
	api.GetProgressNext(rec, req)
	if rec.Code != 200 {
		t.Fatalf("next code=%d body=%s", rec.Code, rec.Body.String())
	}
	var next progressNextResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &next)
	if next.CharacterID == nil || *next.CharacterID != "hira:あ" || next.ReasonCode != learn.NextReasonFirstNotStarted {
		t.Fatalf("fresh next=%+v", next)
	}

	for range 2 {
		d, _ := ls.Attempts().CreateDraft(ctx, learn.CreateDraft{UserID: uid, CharacterID: "hira:あ", LessonID: "lesson:hiragana5"})
		_ = ls.Attempts().SubmitStrokes(ctx, uid, d.ID, []learn.StrokeInput{
			{Width: 2, Color: "#000", Points: []learn.StrokePoint{{X: 1, Y: 1}}},
		}, 300, 300)
		_, _ = ls.Assessments().SaveResult(ctx, uid, learn.SaveAssessment{
			AttemptID: d.ID, Pass: true, Score: 0.9, ScoreKind: recognize.ScoreKindMatch,
			Assessor: learn.AssessorTargetCompare, SetID: recognize.SetIDHiragana5,
		})
	}

	req = attemptSessionReq(t, api, http.MethodGet, "/api/progress/next", uid, "")
	rec = httptest.NewRecorder()
	api.GetProgressNext(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &next)
	if next.CharacterID == nil || *next.CharacterID != "hira:い" {
		t.Fatalf("after あ steady next=%+v", next)
	}

	if _, err := store.ApplyStrokeCreate(uid, 0, "keep-board", "#000000", 2, 1, []db.StrokePoint{{X: 3, Y: 3}}); err != nil {
		t.Fatalf("board stroke: %v", err)
	}
	req = attemptSessionReq(t, api, http.MethodDelete, "/api/practice-data", uid, "")
	rec = httptest.NewRecorder()
	api.ClearPracticeData(rec, req)
	if rec.Code != 200 {
		t.Fatalf("clear code=%d body=%s", rec.Code, rec.Body.String())
	}
	var cleared clearPracticeDataResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &cleared)
	if cleared.AttemptsDeleted < 1 {
		t.Fatalf("clear resp=%+v", cleared)
	}

	req = attemptSessionReq(t, api, http.MethodGet, "/api/attempts", uid, "")
	rec = httptest.NewRecorder()
	api.ListAttempts(rec, req)
	var hist attemptHistoryListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &hist)
	if len(hist.Items) != 0 {
		t.Fatalf("history after clear: %+v", hist.Items)
	}
	board, _ := store.ListStrokesByUser(uid)
	if len(board) != 1 {
		t.Fatalf("board after clear=%d", len(board))
	}
}
