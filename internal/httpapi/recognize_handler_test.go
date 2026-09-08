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

func TestRecognize_BadJSON(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-badjson.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("r1@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader("{"))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "bad_json" {
		t.Fatalf("expected bad_json, got %q", body.Error)
	}
}

func TestRecognize_InvalidDimensions(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-dims.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("r2@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	payload := `{"topN":10,"width":999999,"height":999999}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(payload))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "invalid_dimensions" {
		t.Fatalf("expected invalid_dimensions, got %q", body.Error)
	}
}

func TestRecognize_InvalidTopN(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-topn.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("r3@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	payload := `{"topN":0,"width":300,"height":300}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(payload))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "invalid_top_n" {
		t.Fatalf("expected invalid_top_n, got %q", body.Error)
	}
}

func TestRecognize_HappyPath(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-ok.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("r4@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = store.SaveStroke(userID, "#1d4ed8", 4, 1, []db.StrokePoint{{X: 10, Y: 10}, {X: 50, Y: 10}})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	payload := `{"topN":10,"width":300,"height":300,"boardRev":0}`
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
	if len(out.Candidates) == 0 {
		t.Fatal("expected candidates")
	}
	if out.BoardRev != 0 {
		t.Fatalf("expected boardRev 0, got %d", out.BoardRev)
	}
}

func TestRecognize_PayloadTooLarge(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-big.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)

	userID, err := store.CreateUser("r5@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	big := `{"topN":10,"width":300,"height":300,"pad":"` + strings.Repeat("x", limits.MaxRecognizeBodyBytes) + `"}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(big))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "payload_too_large" && body.Error != "bad_json" {
		t.Fatalf("expected payload_too_large or bad_json, got %q", body.Error)
	}
}

func TestRecognize_RateLimited(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, "test-recognize-rl.db")
	api.Recognizer = recognize.NewSimpleRecognizer()
	api.RecognizeLimiter = limits.NewLimiter(30, 2)

	userID, err := store.CreateUser("r6@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	for i := 0; i < 2; i++ {
		req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
		req.Body = io.NopCloser(strings.NewReader(`{"topN":10,"width":300,"height":300,"boardRev":0}`))
		rec := httptest.NewRecorder()
		api.Recognize(rec, req)
		if rec.Code != 200 {
			t.Fatalf("burst %d: expected 200, got %d body=%s", i, rec.Code, rec.Body.String())
		}
	}
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", userID)
	req.Body = io.NopCloser(strings.NewReader(`{"topN":10,"width":300,"height":300,"boardRev":0}`))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 429 {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "rate_limited" {
		t.Fatalf("got %q", body.Error)
	}
}

