package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
	"github.com/deliium/drawing-board/internal/security"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func TestAttemptE2ELifecycle(t *testing.T) {
	metrics.ResetForTest()
	store, err := db.Open(filepath.Join(t.TempDir(), "attempt-e2e.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.SQL.Close() })

	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	authSvc := auth.NewService(store, sessionStore, false)
	recz, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	api := &API{
		Auth:             authSvc,
		Store:            store,
		Learn:            db.NewLearnStore(store),
		Recognizer:       recz,
		Assessor:         recz,
		RecognizeLimiter: limits.NewLimiter(1000, 1000),
	}

	r := mux.NewRouter()
	r.Handle("/api/lessons/{id}", authSvc.RequireAuth(http.HandlerFunc(api.GetLesson))).Methods(http.MethodGet)
	r.Handle("/api/progress", authSvc.RequireAuth(http.HandlerFunc(api.ListProgress))).Methods(http.MethodGet)
	r.Handle("/api/progress/next", authSvc.RequireAuth(http.HandlerFunc(api.GetProgressNext))).Methods(http.MethodGet)
	r.Handle("/api/practice-data", authSvc.RequireAuth(http.HandlerFunc(api.ClearPracticeData))).Methods(http.MethodDelete)
	r.Handle("/api/attempts", authSvc.RequireAuth(http.HandlerFunc(api.ListAttempts))).Methods(http.MethodGet)
	r.Handle("/api/attempts", authSvc.RequireAuth(http.HandlerFunc(api.CreateAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}", authSvc.RequireAuth(http.HandlerFunc(api.GetAttempt))).Methods(http.MethodGet)
	r.Handle("/api/attempts/{id}/submit", authSvc.RequireAuth(http.HandlerFunc(api.SubmitAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}/assess", authSvc.RequireAuth(http.HandlerFunc(api.AssessAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}/assessment", authSvc.RequireAuth(http.HandlerFunc(api.GetAttemptAssessment))).Methods(http.MethodGet)
	handler := security.CSRF(false, r)

	uid, err := store.CreateUser("e2e@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}

	do := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = bytes.NewReader([]byte(body))
		}
		req := httptest.NewRequest(method, path, reader)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		recSess := httptest.NewRecorder()
		sess, err := authSvc.Sessions.Get(req, "sid")
		if err != nil {
			t.Fatalf("session: %v", err)
		}
		sess.Values["user_id"] = uid
		if err := sess.Save(req, recSess); err != nil {
			t.Fatalf("save session: %v", err)
		}
		csrfToken := "e2e-csrf-token-value-32chars-min!!"
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: csrfToken})
		req.Header.Set(security.CSRFHeaderName, csrfToken)
		for _, c := range recSess.Result().Cookies() {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := do(http.MethodPost, "/api/attempts", `{"characterId":"hira:あ","lessonId":"lesson:hiragana5","clientAttemptId":"e2e-1"}`)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.ID

	submit := `{
		"width":300,"height":300,
		"strokes":[
			{"color":"#000000","width":2,"points":[{"x":60,"y":66},{"x":240,"y":66}]},
			{"color":"#000000","width":2,"points":[{"x":156,"y":36},{"x":156,"y":165},{"x":126,"y":216}]},
			{"color":"#000000","width":2,"points":[{"x":84,"y":144},{"x":165,"y":174},{"x":216,"y":234},{"x":135,"y":264},{"x":90,"y":216}]}
		]
	}`
	rec = do(http.MethodPost, "/api/attempts/"+id+"/submit", submit)
	if rec.Code != 200 {
		t.Fatalf("submit: %d %s", rec.Code, rec.Body.String())
	}

	rec = do(http.MethodPost, "/api/attempts/"+id+"/assess", `{}`)
	if rec.Code != 200 {
		t.Fatalf("assess: %d %s", rec.Code, rec.Body.String())
	}
	var assessed assessmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &assessed)
	if assessed.ScoreKind != recognize.ScoreKindMatch {
		t.Fatalf("scoreKind=%q", assessed.ScoreKind)
	}

	rec = do(http.MethodGet, "/api/attempts/"+id+"/assessment", "")
	if rec.Code != 200 {
		t.Fatalf("get assessment: %d %s", rec.Code, rec.Body.String())
	}

	rec = do(http.MethodPost, "/api/attempts", `{"characterId":"hira:あ","clientAttemptId":"e2e-2"}`)
	if rec.Code != 201 {
		t.Fatalf("retry create: %d %s", rec.Code, rec.Body.String())
	}
	var retry attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &retry)
	if retry.ID == created.ID {
		t.Fatal("retry must create a new attempt row")
	}

	_, err = store.SaveStroke(uid, "#000", 2, 1, []db.StrokePoint{{X: 1, Y: 1}, {X: 9, Y: 9}})
	if err != nil {
		t.Fatal(err)
	}
	rev, _ := store.GetBoardRev(uid)
	if _, err := store.ApplyClear(uid, rev, "e2e-clear"); err != nil {
		t.Fatalf("clear: %v", err)
	}

	rec = do(http.MethodGet, "/api/attempts/"+id+"/assessment", "")
	if rec.Code != 200 {
		t.Fatalf("assessment after board clear: %d %s", rec.Code, rec.Body.String())
	}

	// Progress GET should reflect the assessed attempt (coexistence with curriculum reads).
	rec = do(http.MethodGet, "/api/progress?lessonId=lesson:hiragana5", "")
	if rec.Code != 200 {
		t.Fatalf("progress: %d %s", rec.Code, rec.Body.String())
	}
	var progress progressListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &progress); err != nil {
		t.Fatalf("progress decode: %v", err)
	}
	found := false
	for _, item := range progress.Items {
		if item.CharacterID == "hira:あ" {
			found = true
			if item.AttemptCount < 1 {
				t.Fatalf("progress item=%+v", item)
			}
			t.Logf("progress after assess: status=%s attempts=%d passes=%d", item.Status, item.AttemptCount, item.PassCount)
		}
	}
	if !found {
		t.Fatalf("expected progress for hira:あ, got %+v", progress.Items)
	}

	rec = do(http.MethodGet, "/api/lessons/lesson:hiragana5", "")
	if rec.Code != 200 {
		t.Fatalf("lesson: %d %s", rec.Code, rec.Body.String())
	}
	var lesson lessonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &lesson); err != nil {
		t.Fatalf("lesson decode: %v", err)
	}
	if len(lesson.Characters) != 5 {
		t.Fatalf("lesson characters=%d", len(lesson.Characters))
	}
}
