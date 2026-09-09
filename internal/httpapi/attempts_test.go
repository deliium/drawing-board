package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
	"github.com/deliium/drawing-board/internal/security"
	"github.com/gorilla/mux"
)

func newAttemptTestAPI(t *testing.T, name string) (*API, *db.Store, int64) {
	t.Helper()
	metrics.ResetForTest()
	api, store, _ := newTestAPI(t, filepath.Join(t.TempDir(), name))
	recz, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	api.Recognizer = recz
	api.Assessor = recz
	api.Learn = db.NewLearnStore(store)
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)
	uid, err := store.CreateUser(name+"@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return api, store, uid
}

func attemptSessionReq(t *testing.T, api *API, method, path string, uid int64, body string) *http.Request {
	t.Helper()
	req := sessionRequest(t, api.Auth, method, path, uid)
	if body != "" {
		req.Body = io.NopCloser(strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func withMuxVars(req *http.Request, id string) *http.Request {
	return mux.SetURLVars(req, map[string]string{"id": id})
}

func TestRecognize_TargetRejected(t *testing.T) {
	metrics.ResetForTest()
	api, store, authSvc := newTestAPI(t, filepath.Join(t.TempDir(), "reject-target.db"))
	recz, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("recognizer: %v", err)
	}
	api.Recognizer = recz
	api.Assessor = recz
	api.RecognizeLimiter = limits.NewLimiter(1000, 1000)
	uid, err := store.CreateUser("tgt-reject@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"topN":5,"width":300,"height":300,"boardRev":0,"target":"あ"}`
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/recognize", uid)
	req.Body = io.NopCloser(strings.NewReader(payload))
	rec := httptest.NewRecorder()
	api.Recognize(rec, req)
	if rec.Code != 400 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "use_attempt_api" {
		t.Fatalf("error=%q", body.Error)
	}
}

func TestAttemptCreateUnauthorized(t *testing.T) {
	api, _, _ := newAttemptTestAPI(t, "att-unauth")
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", strings.NewReader(`{"characterId":"hira:あ"}`))
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 401 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestAttemptLifecycleAndIdempotency(t *testing.T) {
	api, store, uid := newAttemptTestAPI(t, "att-life")

	createBody := `{"characterId":"hira:あ","lessonId":"lesson:hiragana5","clientAttemptId":"attempt-client-1"}`
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, createBody)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create code=%d body=%s", rec.Code, rec.Body.String())
	}
	var created attemptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != "draft" || created.Glyph != "あ" {
		t.Fatalf("created=%+v", created)
	}
	id := strconv.FormatInt(created.ID, 10)

	req = attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, createBody)
	rec = httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("idempotent create code=%d", rec.Code)
	}
	var replay attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &replay)
	if replay.ID != created.ID {
		t.Fatalf("replay id=%d want %d", replay.ID, created.ID)
	}

	mismatch := `{"characterId":"hira:い","clientAttemptId":"attempt-client-1"}`
	req = attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, mismatch)
	rec = httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 409 {
		t.Fatalf("mismatch code=%d", rec.Code)
	}

	submit := `{
		"width":300,"height":300,
		"strokes":[
			{"color":"#000000","width":2,"points":[{"x":60,"y":66},{"x":240,"y":66}]},
			{"color":"#000000","width":2,"points":[{"x":156,"y":36},{"x":156,"y":165},{"x":126,"y":216}]},
			{"color":"#000000","width":2,"points":[{"x":84,"y":144},{"x":165,"y":174},{"x":216,"y":234},{"x":135,"y":264},{"x":90,"y":216}]}
		]
	}`
	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/submit", uid, submit), id)
	rec = httptest.NewRecorder()
	api.SubmitAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("submit code=%d body=%s", rec.Code, rec.Body.String())
	}

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/submit", uid, submit), id)
	rec = httptest.NewRecorder()
	api.SubmitAttempt(rec, req)
	if rec.Code != 409 {
		t.Fatalf("second submit code=%d", rec.Code)
	}

	_, _ = store.SaveStroke(uid, "#000", 2, 1, []db.StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	rev, _ := store.GetBoardRev(uid)
	_, _ = store.ApplyClear(uid, rev, "clear-board-1")

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/assess", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AssessAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("assess code=%d body=%s", rec.Code, rec.Body.String())
	}
	var assessed assessmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &assessed)
	if assessed.Status != "assessed" || assessed.ScoreKind != recognize.ScoreKindMatch || assessed.Assessor != "target_compare" {
		t.Fatalf("assessed=%+v", assessed)
	}
	if !assessed.Pass {
		t.Fatalf("gold-ish submit should pass: %+v feedback=%v", assessed, assessed.Feedback)
	}
	if len(assessed.Feedback) > 2 {
		t.Fatalf("feedback len=%d want ≤2", len(assessed.Feedback))
	}
	for _, fb := range assessed.Feedback {
		if fb.Message == "" {
			t.Fatalf("empty feedback message for code=%s", fb.Code)
		}
	}
	if len(assessed.Candidates) == 0 {
		t.Fatal("expected live candidates on assess")
	}

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/assess", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AssessAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("idempotent assess code=%d", rec.Code)
	}
	var assessed2 assessmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &assessed2)
	if assessed2.Pass != assessed.Pass || assessed2.Score != assessed.Score {
		t.Fatalf("replay mismatch %+v vs %+v", assessed2, assessed)
	}

	req = withMuxVars(attemptSessionReq(t, api, http.MethodGet, "/api/attempts/"+id+"/assessment", uid, ""), id)
	rec = httptest.NewRecorder()
	api.GetAttemptAssessment(rec, req)
	if rec.Code != 200 {
		t.Fatalf("get assessment code=%d", rec.Code)
	}
}

func TestAttemptAssessWhileDraft(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "att-draft-assess")
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:い"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := strconv.FormatInt(created.ID, 10)

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/assess", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AssessAttempt(rec, req)
	if rec.Code != 409 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAttemptAbandon(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "att-abandon")
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:う"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := strconv.FormatInt(created.ID, 10)

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/abandon", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AbandonAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/abandon", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AbandonAttempt(rec, req)
	if rec.Code != 409 {
		t.Fatalf("second abandon code=%d", rec.Code)
	}
}

func TestAttemptOtherUserNotFound(t *testing.T) {
	api, store, uid := newAttemptTestAPI(t, "att-other")
	other, err := store.CreateUser("other-att@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:え"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := strconv.FormatInt(created.ID, 10)

	req = withMuxVars(attemptSessionReq(t, api, http.MethodGet, "/api/attempts/"+id, other, ""), id)
	rec = httptest.NewRecorder()
	api.GetAttempt(rec, req)
	if rec.Code != 404 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestAttemptUnknownCharacter(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "att-unknown")
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:ん"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 404 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAttemptSubmitBodyTooLarge(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "att-big")
	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:お"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := strconv.FormatInt(created.ID, 10)

	huge := `{"width":300,"height":300,"strokes":[{"color":"#000","width":2,"points":[{"x":1,"y":1}]}],"pad":"` + strings.Repeat("x", limits.MaxAPIJSONBodyBytes) + `"}`
	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/submit", uid, huge), id)
	rec = httptest.NewRecorder()
	api.SubmitAttempt(rec, req)
	if rec.Code != 400 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "body_too_large" && body.Error != "bad_json" {
		t.Fatalf("error=%q", body.Error)
	}
}

func TestAttemptCSRFRejected(t *testing.T) {
	api, _, uid := newAttemptTestAPI(t, "att-csrf")
	inner := http.HandlerFunc(api.CreateAttempt)
	h := security.CSRF(false, api.Auth.RequireAuth(inner))
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", bytes.NewReader([]byte(`{"characterId":"hira:あ"}`)))
	req.Header.Set("Content-Type", "application/json")
	recSess := httptest.NewRecorder()
	sess, _ := api.Auth.Sessions.Get(req, "sid")
	sess.Values["user_id"] = uid
	_ = sess.Save(req, recSess)
	for _, c := range recSess.Result().Cookies() {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAttemptAssessIncorrectFeedbackMessages(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	api, _, uid := newAttemptTestAPI(t, "att-fb-incorrect")

	req := attemptSessionReq(t, api, http.MethodPost, "/api/attempts", uid, `{"characterId":"hira:あ","clientAttemptId":"fb-incorrect-1"}`)
	rec := httptest.NewRecorder()
	api.CreateAttempt(rec, req)
	if rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("create code=%d body=%s", rec.Code, rec.Body.String())
	}
	var created attemptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := strconv.FormatInt(created.ID, 10)

	// Two strokes for あ (want 3) → count mismatch coaching.
	submit := `{
		"width":300,"height":300,
		"strokes":[
			{"color":"#000000","width":2,"points":[{"x":60,"y":66},{"x":240,"y":66}]},
			{"color":"#000000","width":2,"points":[{"x":156,"y":36},{"x":156,"y":165},{"x":126,"y":216}]}
		]
	}`
	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/submit", uid, submit), id)
	rec = httptest.NewRecorder()
	api.SubmitAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("submit code=%d body=%s", rec.Code, rec.Body.String())
	}

	req = withMuxVars(attemptSessionReq(t, api, http.MethodPost, "/api/attempts/"+id+"/assess", uid, `{}`), id)
	rec = httptest.NewRecorder()
	api.AssessAttempt(rec, req)
	if rec.Code != 200 {
		t.Fatalf("assess code=%d body=%s", rec.Code, rec.Body.String())
	}
	var assessed assessmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &assessed)
	if assessed.Pass {
		t.Fatalf("expected non-pass: %+v", assessed)
	}
	if assessed.ScoreKind != recognize.ScoreKindMatch {
		t.Fatalf("scoreKind=%q", assessed.ScoreKind)
	}
	if len(assessed.Feedback) == 0 || len(assessed.Feedback) > 2 {
		t.Fatalf("feedback=%#v", assessed.Feedback)
	}
	if assessed.Feedback[0].Code != recognize.CodeStrokeCountMismatch {
		t.Fatalf("primary code=%s want stroke_count_mismatch", assessed.Feedback[0].Code)
	}
	for _, fb := range assessed.Feedback {
		if strings.TrimSpace(fb.Message) == "" {
			t.Fatalf("empty message for %s", fb.Code)
		}
	}
}
