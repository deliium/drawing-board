package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	CSRFCookieName = "csrf"
	CSRFHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

// NewCSRFToken returns a high-entropy base64url token (≥32 random bytes).
func NewCSRFToken() (string, error) {
	buf := make([]byte, csrfTokenBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		secLogf("ERROR", "[csrf] RNG failed: %v", err)
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// SetCSRFCookie writes the non-HttpOnly csrf cookie with production-aligned Secure/SameSite flags.
func SetCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

// EnsureCSRFCookie returns the existing csrf cookie value or issues a new one.
func EnsureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool) (string, error) {
	if c, err := r.Cookie(CSRFCookieName); err == nil && c.Value != "" {
		secLogf("DEBUG", "[csrf] ensure reuse existing cookie")
		return c.Value, nil
	}
	token, err := NewCSRFToken()
	if err != nil {
		return "", err
	}
	SetCSRFCookie(w, token, secure)
	secLogf("DEBUG", "[csrf] ensure issued new cookie")
	return token, nil
}

// TokensEqual compares cookie and header tokens in constant time.
func TokensEqual(cookieVal, headerVal string) bool {
	if cookieVal == "" || headerVal == "" {
		return false
	}
	a := []byte(cookieVal)
	b := []byte(headerVal)
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

type csrfErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CSRF protects POST /api/* with double-submit cookie verification.
// Safe GET under /api/ ensures a csrf cookie is present.
// /healthz and /ws are not under this rule when paths do not match.
func CSRF(secure bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead:
			if _, err := EnsureCSRFCookie(w, r, secure); err != nil {
				secLogf("ERROR", "[csrf] ensure failed path=%s: %v", path, err)
				writeCSRFError(w, http.StatusInternalServerError, "csrf_rejected", "Something went wrong. Try again.")
				return
			}
			next.ServeHTTP(w, r)
			return
		case http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		case http.MethodPost:
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || cookie.Value == "" {
				secLogf("DEBUG", "[csrf] reject reason=missing_cookie path=%s", path)
				writeCSRFError(w, http.StatusForbidden, "csrf_rejected", "CSRF token missing or invalid. Refresh and try again.")
				return
			}
			header := strings.TrimSpace(r.Header.Get(CSRFHeaderName))
			if header == "" {
				secLogf("DEBUG", "[csrf] reject reason=missing_header path=%s", path)
				writeCSRFError(w, http.StatusForbidden, "csrf_rejected", "CSRF token missing or invalid. Refresh and try again.")
				return
			}
			if !TokensEqual(cookie.Value, header) {
				secLogf("DEBUG", "[csrf] reject reason=mismatch path=%s", path)
				writeCSRFError(w, http.StatusForbidden, "csrf_rejected", "CSRF token missing or invalid. Refresh and try again.")
				return
			}
			secLogf("DEBUG", "[csrf] ok path=%s", path)
			next.ServeHTTP(w, r)
			return
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func writeCSRFError(w http.ResponseWriter, code int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(csrfErrorBody{Error: errCode, Message: message})
}

// IssueCSRFHandler returns GET /api/csrf that ensures a token cookie and returns {csrf}.
func IssueCSRFHandler(secure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := EnsureCSRFCookie(w, r, secure)
		if err != nil {
			secLogf("ERROR", "[csrf] IssueCSRFHandler failed: %v", err)
			writeCSRFError(w, http.StatusInternalServerError, "csrf_rejected", "Something went wrong. Try again.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"csrf": token})
	}
}
