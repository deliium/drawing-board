package security

import (
	"net/http"
	"strings"
)

// CORS wraps next with an exact-origin allowlist policy.
// Allowlisted Origin gets ACAO + credentials; disallowed Origin never receives reflecting ACAO.
// OPTIONS for disallowed Origin returns 403; allowlisted OPTIONS returns 204 without hitting next.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	origins := append([]string(nil), allowedOrigins...)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		w.Header().Add("Vary", "Origin")

		if origin == "" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if !OriginAllowed(origins, origin) {
			secLogf("DEBUG", "[cors] deny origin=%s reason=not_allowlisted", origin)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		secLogf("DEBUG", "[cors] allow origin=%s", origin)
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
