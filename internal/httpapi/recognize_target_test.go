package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
)

func TestRecognize_UnsupportedTarget(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-badtarget.db")
	recz, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	api.Recognizer = recz
	api.Assessor = recz
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("tgt1@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = store.SaveStroke(userID, "#1d4ed8", 4, 1, []db.StrokePoint{{X: 10, Y: 10}, {X: 50, Y: 10}})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	payload := `{"topN":10,"width":300,"height":300,"boardRev":0,"target":"漢"}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(payload))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "unsupported_target" {
		t.Fatalf("expected unsupported_target, got %q", body.Error)
	}
}

func TestRecognize_TargetMode_Pass(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-targetok.db")
	recz, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	api.Recognizer = recz
	api.Assessor = recz
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("tgt2@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Use い template strokes (two strokes) scaled into canvas space.
	strokes := [][]db.StrokePoint{
		{{X: 28, Y: 22}, {X: 22, Y: 55}, {X: 30, Y: 82}},
		{{X: 62, Y: 28}, {X: 72, Y: 55}, {X: 68, Y: 80}},
	}
	for _, pts := range strokes {
		if _, err := store.SaveStroke(userID, "#1d4ed8", 4, 1, pts); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	payload := `{"topN":5,"width":300,"height":300,"boardRev":0,"target":"い"}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(payload))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out RecognizeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ScoreKind != recognize.ScoreKindMatch {
		t.Fatalf("scoreKind=%q", out.ScoreKind)
	}
	if out.Assessment == nil || !out.Assessment.Pass {
		t.Fatalf("expected assessment pass, got %+v", out.Assessment)
	}
	if out.Assessment.Target != "い" {
		t.Fatalf("target=%q", out.Assessment.Target)
	}
}
