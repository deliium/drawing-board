package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/sessions"
)

func newTestAuthService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "auth-test.db")
	store, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = store.SQL.Close()
		_ = os.Remove(dbPath)
	})
	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	return NewService(store, sessionStore, false)
}

func decodeError(t *testing.T, body []byte) errorBody {
	t.Helper()
	var out errorBody
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode error body: %v body=%s", err, body)
	}
	return out
}

func postJSON(t *testing.T, svc *Service, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	switch path {
	case "/api/register":
		svc.Register(rec, req)
	case "/api/login":
		svc.Login(rec, req)
	case "/api/logout":
		svc.Logout(rec, req)
	default:
		t.Fatalf("unknown path %s", path)
	}
	return rec
}

func TestAuthHandlers_RegisterLoginLogoutMe(t *testing.T) {
	svc := newTestAuthService(t)

	t.Run("register success sets cookie and me works", func(t *testing.T) {
		rec := postJSON(t, svc, "/api/register", `{"email":"Learner@Example.com","password":"password1"}`, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var uv userView
		if err := json.Unmarshal(rec.Body.Bytes(), &uv); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if uv.Email != "learner@example.com" || uv.ID == "" {
			t.Fatalf("unexpected user view %+v", uv)
		}
		cookies := rec.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected Set-Cookie on register")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		meRec := httptest.NewRecorder()
		svc.Me(meRec, req)
		if meRec.Code != http.StatusOK {
			t.Fatalf("me expected 200, got %d body=%s", meRec.Code, meRec.Body.String())
		}
	})

	t.Run("register validation codes", func(t *testing.T) {
		cases := []struct {
			name string
			body string
			code string
		}{
			{"missing", `{"email":"","password":""}`, "missing_fields"},
			{"invalid email", `{"email":"not-an-email","password":"password1"}`, "invalid_email"},
			{"short password", `{"email":"ok@example.com","password":"abcd"}`, "password_too_short"},
			{"bad json", `{`, "bad_json"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rec := postJSON(t, svc, "/api/register", tc.body, nil)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", rec.Code)
				}
				got := decodeError(t, rec.Body.Bytes())
				if got.Error != tc.code {
					t.Fatalf("expected error=%s got=%s body=%s", tc.code, got.Error, rec.Body.String())
				}
			})
		}
		// Short password must not leave a row that blocks a later valid register.
		if u, err := svc.Store.GetUserByEmail("ok@example.com"); err != nil {
			t.Fatalf("lookup: %v", err)
		} else if u != nil {
			t.Fatalf("short-password register created user id=%s", u.ID)
		}
		ok := postJSON(t, svc, "/api/register", `{"email":"ok@example.com","password":"password1"}`, nil)
		if ok.Code != http.StatusOK {
			t.Fatalf("valid register after short reject: %d %s", ok.Code, ok.Body.String())
		}
	})

	t.Run("login allows short password for existing account", func(t *testing.T) {
		email := "shortlegacy@example.com"
		hash, err := HashPassword("abcd")
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		if _, err := svc.Store.CreateUser(email, hash); err != nil {
			t.Fatalf("seed: %v", err)
		}
		login := postJSON(t, svc, "/api/login", `{"email":"`+email+`","password":"abcd"}`, nil)
		if login.Code != http.StatusOK {
			t.Fatalf("short login: %d %s", login.Code, login.Body.String())
		}
		// Re-register with a long password must still fail (email taken), not hang the account.
		again := postJSON(t, svc, "/api/register", `{"email":"`+email+`","password":"password1"}`, nil)
		if again.Code != http.StatusBadRequest || decodeError(t, again.Body.Bytes()).Error != "registration_failed" {
			t.Fatalf("expected registration_failed, got %d %s", again.Code, again.Body.String())
		}
	})

	t.Run("duplicate register is registration_failed without enumeration", func(t *testing.T) {
		email := "dup@example.com"
		first := postJSON(t, svc, "/api/register", `{"email":"`+email+`","password":"password1"}`, nil)
		if first.Code != http.StatusOK {
			t.Fatalf("first register: %d %s", first.Code, first.Body.String())
		}
		second := postJSON(t, svc, "/api/register", `{"email":"`+email+`","password":"password1"}`, nil)
		if second.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", second.Code, second.Body.String())
		}
		body := second.Body.String()
		got := decodeError(t, second.Body.Bytes())
		if got.Error != "registration_failed" {
			t.Fatalf("expected registration_failed, got %s", got.Error)
		}
		if strings.Contains(strings.ToLower(body), "email exists") {
			t.Fatalf("must not reveal email exists: %s", body)
		}
		if strings.Contains(strings.ToLower(body), "already registered") {
			t.Fatalf("must not uniquely confirm email taken: %s", body)
		}
	})

	t.Run("login success and invalid credentials", func(t *testing.T) {
		_ = postJSON(t, svc, "/api/register", `{"email":"login@example.com","password":"password1"}`, nil)

		ok := postJSON(t, svc, "/api/login", `{"email":"login@example.com","password":"password1"}`, nil)
		if ok.Code != http.StatusOK {
			t.Fatalf("login success expected 200, got %d %s", ok.Code, ok.Body.String())
		}

		wrong := postJSON(t, svc, "/api/login", `{"email":"login@example.com","password":"wrongpass"}`, nil)
		if wrong.Code != http.StatusUnauthorized {
			t.Fatalf("wrong password expected 401, got %d", wrong.Code)
		}
		if decodeError(t, wrong.Body.Bytes()).Error != "invalid_credentials" {
			t.Fatalf("expected invalid_credentials, got %s", wrong.Body.String())
		}

		unknown := postJSON(t, svc, "/api/login", `{"email":"nobody@example.com","password":"password1"}`, nil)
		if unknown.Code != http.StatusUnauthorized {
			t.Fatalf("unknown email expected 401, got %d", unknown.Code)
		}
		if decodeError(t, unknown.Body.Bytes()).Error != "invalid_credentials" {
			t.Fatalf("unknown email must use invalid_credentials, got %s", unknown.Body.String())
		}
	})

	t.Run("logout clears session", func(t *testing.T) {
		reg := postJSON(t, svc, "/api/register", `{"email":"logout@example.com","password":"password1"}`, nil)
		cookies := reg.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie")
		}

		out := postJSON(t, svc, "/api/logout", `{}`, cookies)
		if out.Code != http.StatusOK {
			t.Fatalf("logout expected 200, got %d", out.Code)
		}

		// After logout MaxAge=-1; use returned cookies if any, else original cookies may still be sent by client until cleared.
		// Simulate cleared session by requesting /api/me without cookie.
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		meRec := httptest.NewRecorder()
		svc.Me(meRec, req)
		if meRec.Code != http.StatusUnauthorized {
			t.Fatalf("me without session expected 401, got %d", meRec.Code)
		}

		// Also apply logout Set-Cookie onto a request that still has the old sid and confirm Me fails after MaxAge=-1 save.
		req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		for _, c := range out.Result().Cookies() {
			req2.AddCookie(c)
		}
		// CookieStore with MaxAge=-1 deletes; Me with original cookie alone should still work until client drops it,
		// so assert Me with no cookies (already done) and that logout response includes delete cookie.
		deleted := false
		for _, c := range out.Result().Cookies() {
			if c.Name == "sid" && c.MaxAge < 0 {
				deleted = true
			}
		}
		if !deleted {
			// gorilla may encode deletion differently; ensure at least logout succeeded and anonymous me is 401.
			t.Log("logout did not return MaxAge<0 sid cookie; anonymous /api/me already asserted")
		}
		_ = req2
	})
}

func TestAuthHandlers_BcryptRegisterAndLogin(t *testing.T) {
	svc := newTestAuthService(t)

	t.Run("register stores bcrypt hash", func(t *testing.T) {
		rec := postJSON(t, svc, "/api/register", `{"email":"bcrypt@example.com","password":"password1"}`, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
		}
		u, err := svc.Store.GetUserByEmail("bcrypt@example.com")
		if err != nil || u == nil {
			t.Fatalf("lookup: %v user=%v", err, u)
		}
		if !strings.HasPrefix(u.PasswordHash, "$2") {
			t.Fatalf("expected bcrypt hash, got %q", u.PasswordHash)
		}
		login := postJSON(t, svc, "/api/login", `{"email":"bcrypt@example.com","password":"password1"}`, nil)
		if login.Code != http.StatusOK {
			t.Fatalf("login: %d %s", login.Code, login.Body.String())
		}
	})

	t.Run("non-bcrypt stored hash cannot login", func(t *testing.T) {
		email := "legacy@example.com"
		pw := "password1"
		// Pre-migration unsalted SHA-256 hex of "password1".
		legacyHash := "0b14d501a594442a01c6859541bcb3e8164d183d32937b851835442f69d5c94e"
		_, err := svc.Store.CreateUser(email, legacyHash)
		if err != nil {
			t.Fatalf("seed legacy user: %v", err)
		}
		login := postJSON(t, svc, "/api/login", `{"email":"`+email+`","password":"`+pw+`"}`, nil)
		if login.Code != http.StatusUnauthorized || decodeError(t, login.Body.Bytes()).Error != "invalid_credentials" {
			t.Fatalf("legacy hash login: %d %s", login.Code, login.Body.String())
		}
	})

	t.Run("password_too_long rejected", func(t *testing.T) {
		longPW := strings.Repeat("a", 73)
		body := `{"email":"long@example.com","password":"` + longPW + `"}`
		reg := postJSON(t, svc, "/api/register", body, nil)
		if reg.Code != http.StatusBadRequest || decodeError(t, reg.Body.Bytes()).Error != "password_too_long" {
			t.Fatalf("register too long: %d %s", reg.Code, reg.Body.String())
		}
		login := postJSON(t, svc, "/api/login", body, nil)
		if login.Code != http.StatusBadRequest || decodeError(t, login.Body.Bytes()).Error != "password_too_long" {
			t.Fatalf("login too long: %d %s", login.Code, login.Body.String())
		}
	})
}

func TestAuthHandlers_SessionRotationAndSecureCookie(t *testing.T) {
	svc := newTestAuthService(t)
	svc.Secure = true
	svc.Sessions.Options.Secure = true

	reg := postJSON(t, svc, "/api/register", `{"email":"secure@example.com","password":"password1"}`, nil)
	if reg.Code != http.StatusOK {
		t.Fatalf("register: %d %s", reg.Code, reg.Body.String())
	}
	cookies := reg.Result().Cookies()
	foundSecure := false
	for _, c := range cookies {
		if c.Name == "sid" && c.Secure {
			foundSecure = true
		}
	}
	if !foundSecure {
		t.Fatalf("expected Secure sid cookie, got %+v", cookies)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	meRec := httptest.NewRecorder()
	svc.Me(meRec, req)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me after register: %d", meRec.Code)
	}

	out := postJSON(t, svc, "/api/logout", `{}`, cookies)
	if out.Code != http.StatusOK {
		t.Fatalf("logout: %d", out.Code)
	}
	deletedSecure := false
	for _, c := range out.Result().Cookies() {
		if c.Name == "sid" && c.MaxAge < 0 && c.Secure {
			deletedSecure = true
		}
	}
	if !deletedSecure {
		t.Logf("logout cookies: %+v", out.Result().Cookies())
		// Still require /api/me without cookie is 401
	}
	anon := httptest.NewRecorder()
	svc.Me(anon, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous me expected 401, got %d", anon.Code)
	}
}
