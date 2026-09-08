package security

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// DefaultDevOrigins returns the Vite loopback pair used when ALLOWED_ORIGINS is unset in development.
func DefaultDevOrigins() []string {
	return []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}
}

// IsProduction reports whether APP_ENV is production (case-insensitive).
func IsProduction(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "production")
}

// ParseAllowedOrigins parses a comma-separated exact-origin allowlist.
// Wildcards, blank entries, non-http(s) schemes, and path/query/fragment are rejected.
func ParseAllowedOrigins(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("ALLOWED_ORIGINS is empty")
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for i, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			return nil, fmt.Errorf("ALLOWED_ORIGINS entry %d is blank", i+1)
		}
		if strings.Contains(entry, "*") {
			return nil, fmt.Errorf("ALLOWED_ORIGINS does not support wildcards (entry %d)", i+1)
		}
		normalized, err := normalizeOriginString(entry)
		if err != nil {
			return nil, fmt.Errorf("ALLOWED_ORIGINS entry %d: %w", i+1, err)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ALLOWED_ORIGINS is empty after parsing")
	}
	debugf("[security.ParseAllowedOrigins] count=%d", len(out))
	return out, nil
}

// ResolveAllowedOrigins applies environment-aware defaults and validation.
// Development with empty raw → DefaultDevOrigins.
// Production with empty/invalid raw → error (caller should FATAL).
func ResolveAllowedOrigins(appEnv, raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if IsProduction(appEnv) {
			return nil, fmt.Errorf("ALLOWED_ORIGINS is required when APP_ENV=production")
		}
		origins := DefaultDevOrigins()
		debugf("[security.ResolveAllowedOrigins] mode=development using defaults count=%d", len(origins))
		return origins, nil
	}
	origins, err := ParseAllowedOrigins(raw)
	if err != nil {
		return nil, err
	}
	mode := "development"
	if IsProduction(appEnv) {
		mode = "production"
	}
	debugf("[security.ResolveAllowedOrigins] mode=%s count=%d", mode, len(origins))
	return origins, nil
}

// OriginAllowed reports whether origin exactly matches an allowlisted entry after normalization.
func OriginAllowed(origins []string, origin string) bool {
	normalized, err := normalizeOriginString(origin)
	if err != nil {
		return false
	}
	for _, allowed := range origins {
		if allowed == normalized {
			return true
		}
	}
	return false
}

func normalizeOriginString(origin string) (string, error) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", fmt.Errorf("origin is empty")
	}
	if strings.Contains(origin, "*") {
		return "", fmt.Errorf("wildcard origins are not allowed")
	}
	u, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("invalid origin URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("origin scheme must be http or https (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("origin host is required")
	}
	if u.User != nil {
		return "", fmt.Errorf("origin must not include userinfo")
	}
	if u.Opaque != "" {
		return "", fmt.Errorf("origin must not include opaque data")
	}
	path := u.EscapedPath()
	if path != "" && path != "/" {
		return "", fmt.Errorf("origin must not include a path")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("origin must not include query or fragment")
	}
	host := strings.ToLower(u.Host)
	return scheme + "://" + host, nil
}

// debugf emits DEBUG lines unless LOG_LEVEL suppresses them.
func debugf(format string, args ...interface{}) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error", "warn", "info":
		return
	}
	// Re-use standard log via a tiny indirection to keep package free of init.
	secLogf("DEBUG", format, args...)
}
