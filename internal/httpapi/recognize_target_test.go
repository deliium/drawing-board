package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
)

func TestRecognize_UnsupportedTarget(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, filepath.Join(t.TempDir(), "test-recognize-badtarget.db"))
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
	if body.Error != "use_attempt_api" {
		t.Fatalf("expected use_attempt_api, got %q", body.Error)
	}
}

func TestRecognize_TargetMode_Rejected(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, filepath.Join(t.TempDir(), "test-recognize-targetok.db"))
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

	payload := `{"topN":5,"width":300,"height":300,"boardRev":0,"target":"い"}`
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
	if body.Error != "use_attempt_api" {
		t.Fatalf("expected use_attempt_api, got %q", body.Error)
	}
}
