package security

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSAllowlistedOrigin(t *testing.T) {
	t.Parallel()
	origins := []string{"http://localhost:5173"}
	h := CORS(origins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("ACAO=%q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("ACAC=%q", got)
	}
	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowHeaders, "X-CSRF-Token") {
		t.Fatalf("Allow-Headers missing X-CSRF-Token: %q", allowHeaders)
	}
	if strings.Contains(rec.Header().Get("Access-Control-Allow-Origin"), "*") {
		t.Fatal("wildcard ACAO must not appear")
	}
}

func TestCORSDisallowedPreflight(t *testing.T) {
	t.Parallel()
	origins := []string{"http://localhost:5173"}
	h := CORS(origins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for disallowed OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("must not reflect Origin, got ACAO=%q", got)
	}
}

func TestCSRFRejectMissingAndMismatch(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	h := CSRF(false, inner)

	t.Run("missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/logout", strings.NewReader("{}"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
		var body csrfErrorBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error != "csrf_rejected" {
			t.Fatalf("error=%q", body.Error)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/logout", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "cookie-token"})
		req.Header.Set(CSRFHeaderName, "header-token")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("valid", func(t *testing.T) {
		token := "matching-token-value-32chars!!"
		req := httptest.NewRequest(http.MethodPost, "/api/logout", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})
		req.Header.Set(CSRFHeaderName, token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("get ensures cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		found := false
		for _, c := range rec.Result().Cookies() {
			if c.Name == CSRFCookieName && c.Value != "" && !c.HttpOnly {
				found = true
			}
		}
		if !found {
			t.Fatal("expected non-HttpOnly csrf cookie")
		}
	})
}

func TestTokensEqual(t *testing.T) {
	t.Parallel()
	if !TokensEqual("abc", "abc") {
		t.Fatal("expected equal")
	}
	if TokensEqual("abc", "abd") {
		t.Fatal("expected unequal")
	}
	if TokensEqual("", "") {
		t.Fatal("empty must not equal")
	}
}
