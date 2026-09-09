package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deliium/drawing-board/internal/features"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/gorilla/mux"
)

func TestFeaturePracticeOffReturns404(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	metrics.ResetForTest()
	api, _, uid := newAttemptTestAPI(t, "feat-practice")
	off := features.Flags{Practice: false, Progress: false, Review: false, Audio: false}
	api.Features = &off

	req := attemptSessionReq(t, api, http.MethodGet, "/api/lessons/lesson:hiragana5", uid, "")
	req = mux.SetURLVars(req, map[string]string{"id": "lesson:hiragana5"})
	rec := httptest.NewRecorder()
	api.RequireFeature(false, "practice", api.GetLesson).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d body=%s", rec.Code, rec.Body.String())
	}
	if metrics.Get("practice_feature_disabled_total") < 1 {
		t.Fatalf("expected feature_disabled metric increment")
	}
	t.Logf("FEATURE_PRACTICE=0 → 404 lessons")
}

func TestFeatureProgressOffBlocksHistoryAndClear(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	metrics.ResetForTest()
	api, _, uid := newAttemptTestAPI(t, "feat-progress")
	flags := features.Flags{Practice: true, Progress: false, Review: false, Audio: true}
	api.Features = &flags

	req := attemptSessionReq(t, api, http.MethodGet, "/api/attempts", uid, "")
	rec := httptest.NewRecorder()
	api.RequireFeature(false, "progress", api.ListAttempts).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("list attempts want 404 got %d", rec.Code)
	}

	req2 := attemptSessionReq(t, api, http.MethodDelete, "/api/practice-data", uid, "")
	rec2 := httptest.NewRecorder()
	api.RequireFeature(false, "progress", api.ClearPracticeData).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("clear want 404 got %d", rec2.Code)
	}
	t.Logf("FEATURE_PROGRESS=0 → history/clear 404")
}

func TestGetFeaturesReflectsFlags(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "feat-get")
	flags := features.Flags{Practice: true, Progress: true, Review: false, Audio: false}
	api.Features = &flags

	req := attemptSessionReq(t, api, http.MethodGet, "/api/features", uid, "")
	rec := httptest.NewRecorder()
	api.GetFeatures(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var got features.Flags
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Practice != true || got.Progress != true || got.Review != false || got.Audio != false {
		t.Fatalf("unexpected flags %+v", got)
	}
}

func TestProgressNextOmitsReviewWhenFlagOff(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, _, uid := newAttemptTestAPI(t, "feat-next")
	flags := features.Flags{Practice: true, Progress: true, Review: false, Audio: true}
	api.Features = &flags

	req := attemptSessionReq(t, api, http.MethodGet, "/api/progress/next?lessonId=lesson:hiragana5", uid, "")
	rec := httptest.NewRecorder()
	api.GetProgressNext(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if v, ok := body["dueAt"]; ok && v != nil {
		t.Fatalf("dueAt should be omitted when review off: %v", v)
	}
	if v, ok := body["reviewBox"]; ok && v != nil {
		t.Fatalf("reviewBox should be omitted when review off: %v", v)
	}
	t.Logf("FEATURE_REVIEW=0 next reason=%v", body["reasonCode"])
}

func TestFeatureAudioStripsAudioRef(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "feat-audio")
	flags := features.Flags{Practice: true, Progress: true, Review: true, Audio: false}
	api.Features = &flags

	req := attemptSessionReq(t, api, http.MethodGet, "/api/lessons/lesson:hiragana5", uid, "")
	req = mux.SetURLVars(req, map[string]string{"id": "lesson:hiragana5"})
	rec := httptest.NewRecorder()
	api.GetLesson(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp lessonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, ch := range resp.Characters {
		var pron map[string]any
		if err := json.Unmarshal(ch.Pronunciation, &pron); err != nil {
			t.Fatal(err)
		}
		if _, ok := pron["audioRef"]; ok {
			t.Fatalf("audioRef should be stripped when FEATURE_AUDIO=0 for %s", ch.ID)
		}
	}
}
