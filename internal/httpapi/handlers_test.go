package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/sessions"
)

func TestNewAPI(t *testing.T) {
	authService := &auth.Service{}
	store := &db.Store{}

	api := &API{
		Auth:  authService,
		Store: store,
	}

	if api.Auth != authService {
		t.Fatal("Auth should be set correctly")
	}

	if api.Store != store {
		t.Fatal("Store should be set correctly")
	}
}

func newTestAPI(t *testing.T, dbPath string) (*API, *db.Store, *auth.Service) {
	t.Helper()
	store, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = store.SQL.Close()
		_ = os.Remove(dbPath)
	})
	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	authSvc := &auth.Service{Store: store, Sessions: sessionStore}
	return &API{Auth: authSvc, Store: store}, store, authSvc
}

func sessionRequest(t *testing.T, authSvc *auth.Service, method, path string, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	// Create session cookie via startSession path: Login writes cookie; mimic by saving session.
	sess, err := authSvc.Sessions.Get(req, "sid")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	sess.Values["user_id"] = userID
	if err := sess.Save(req, rec); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func TestListStrokes_Unauthorized(t *testing.T) {
	api, _, _ := newTestAPI(t, "test_http_list_unauth.db")
	req := httptest.NewRequest(http.MethodGet, "/api/strokes", nil)
	rec := httptest.NewRecorder()
	api.ListStrokes(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 unauthorized, got %d", rec.Code)
	}
}

func TestClearStrokes_Unauthorized(t *testing.T) {
	api, _, _ := newTestAPI(t, "test_http_clear_unauth.db")
	req := httptest.NewRequest(http.MethodPost, "/api/strokes/clear", nil)
	rec := httptest.NewRecorder()
	api.ClearStrokes(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 unauthorized, got %d", rec.Code)
	}
}

func TestDeleteStroke_Unauthorized(t *testing.T) {
	api, _, _ := newTestAPI(t, "test_http_delete_unauth.db")
	req := httptest.NewRequest(http.MethodPost, "/api/strokes/delete?id=1", nil)
	rec := httptest.NewRecorder()
	api.DeleteStroke(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 unauthorized, got %d", rec.Code)
	}
}

func TestDeleteStroke_InvalidUUID(t *testing.T) {
	api, store, authSvc := newTestAPI(t, "test_http_delete_bad_uuid.db")
	uid, err := store.CreateUser("del-bad-uuid@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/strokes/delete?id=not-a-uuid", uid)
	rec := httptest.NewRecorder()
	api.DeleteStroke(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "invalid_input" {
		t.Fatalf("expected invalid_input, got %+v", body)
	}
}

func TestDeleteStroke_UnknownUUID_NotFound(t *testing.T) {
	api, store, authSvc := newTestAPI(t, "test_http_delete_missing.db")
	uid, err := store.CreateUser("del-missing@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	missing := "00000000-0000-4000-8000-000000000099"
	revBefore, err := store.GetBoardRev(uid)
	if err != nil {
		t.Fatalf("rev: %v", err)
	}
	req := sessionRequest(t, authSvc, http.MethodPost, "/api/strokes/delete?id="+missing, uid)
	rec := httptest.NewRecorder()
	api.DeleteStroke(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body deleteStrokeNotFoundBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "not_found" || body.Deleted || body.BoardRev != revBefore+1 {
		t.Fatalf("unexpected body: %+v want boardRev=%d", body, revBefore+1)
	}
}

func TestClearAndDelete_OnlyTouchCallerData(t *testing.T) {
	api, store, authSvc := newTestAPI(t, "test_http_isolation.db")

	userA, err := store.CreateUser("a@example.com", "hash")
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	userB, err := store.CreateUser("b@example.com", "hash")
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	idA, err := store.SaveStroke(userA, "#000", 2, 1, []db.StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	idB, err := store.SaveStroke(userB, "#111", 3, 2, []db.StrokePoint{{X: 3, Y: 3}, {X: 4, Y: 4}})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}

	t.Run("delete own stroke leaves other user intact", func(t *testing.T) {
		req := sessionRequest(t, authSvc, http.MethodPost, "/api/strokes/delete?id="+idA, userA)
		rec := httptest.NewRecorder()
		api.DeleteStroke(rec, req)
		if rec.Code != 200 {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["deleted"] != true {
			t.Fatalf("expected deleted=true, got %v", body)
		}
		strokesB, err := store.ListStrokesByUser(userB)
		if err != nil {
			t.Fatalf("list B: %v", err)
		}
		if len(strokesB) != 1 || strokesB[0].ID != idB {
			t.Fatalf("user B data must remain after A deletes own stroke; got %+v", strokesB)
		}
	})

	t.Run("delete foreign stroke returns not_found and advances caller boardRev", func(t *testing.T) {
		revBefore, err := store.GetBoardRev(userA)
		if err != nil {
			t.Fatalf("rev before: %v", err)
		}
		req := sessionRequest(t, authSvc, http.MethodPost, "/api/strokes/delete?id="+idB, userA)
		rec := httptest.NewRecorder()
		api.DeleteStroke(rec, req)
		if rec.Code != 404 {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
		var body deleteStrokeNotFoundBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Error != "not_found" || body.Deleted || body.ID != idB {
			t.Fatalf("unexpected not_found body: %+v", body)
		}
		if body.BoardRev != revBefore+1 {
			t.Fatalf("boardRev want %d got %d", revBefore+1, body.BoardRev)
		}
		strokesB, err := store.ListStrokesByUser(userB)
		if err != nil || len(strokesB) != 1 || strokesB[0].ID != idB {
			t.Fatalf("user B stroke must remain; got %+v err=%v", strokesB, err)
		}
	})

	t.Run("clear only clears caller strokes", func(t *testing.T) {
		// Ensure A has a stroke again, B still has idB
		_, err := store.SaveStroke(userA, "#000", 2, 3, []db.StrokePoint{{X: 5, Y: 5}, {X: 6, Y: 6}})
		if err != nil {
			t.Fatalf("save A again: %v", err)
		}
		req := sessionRequest(t, authSvc, http.MethodPost, "/api/strokes/clear", userA)
		rec := httptest.NewRecorder()
		api.ClearStrokes(rec, req)
		if rec.Code != 200 {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		strokesA, err := store.ListStrokesByUser(userA)
		if err != nil {
			t.Fatalf("list A: %v", err)
		}
		if len(strokesA) != 0 {
			t.Fatalf("expected user A cleared, got %d strokes", len(strokesA))
		}
		strokesB, err := store.ListStrokesByUser(userB)
		if err != nil {
			t.Fatalf("list B: %v", err)
		}
		if len(strokesB) != 1 {
			t.Fatalf("clear must not touch user B; expected 1 stroke, got %d", len(strokesB))
		}
	})

	t.Run("list returns only caller strokes", func(t *testing.T) {
		req := sessionRequest(t, authSvc, http.MethodGet, "/api/strokes", userB)
		rec := httptest.NewRecorder()
		api.ListStrokes(rec, req)
		if rec.Code != 200 {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var out StrokesListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Strokes) != 1 || out.Strokes[0].ID != idB {
			t.Fatalf("list must return only user B stroke id=%s, got %+v", idB, out)
		}
	})
}
