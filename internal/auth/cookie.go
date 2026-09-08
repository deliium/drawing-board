package auth

import (
	"fmt"
	"strings"
)

// Forbidden production cookie-key sentinels (defaults / README examples).
const (
	CookieKeyDefaultSentinel = "change-me-please-32-bytes-min"
	CookieKeyREADMESentinel  = "please-change-this-32-bytes-min"
	MinCookieKeyBytes        = 32
)

// ProductionSecureMode is true when APP_ENV=production (case-insensitive) or
// COOKIE_SECURE is true/1/yes.
func ProductionSecureMode(appEnv, cookieSecure string) bool {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(cookieSecure)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// IsWeakCookieKey reports empty, short, or known sentinel keys.
func IsWeakCookieKey(key string) bool {
	if key == "" || len(key) < MinCookieKeyBytes {
		return true
	}
	if key == CookieKeyDefaultSentinel || key == CookieKeyREADMESentinel {
		return true
	}
	return false
}

// ValidateCookieKey fails closed in production-secure mode when the key is weak.
// In non-secure mode it always returns nil (caller should WARN separately).
func ValidateCookieKey(key string, secureMode bool) error {
	if !secureMode {
		return nil
	}
	if key == "" {
		return fmt.Errorf("COOKIE_KEY is required when production-secure cookies are enabled")
	}
	if len(key) < MinCookieKeyBytes {
		return fmt.Errorf("COOKIE_KEY must be at least %d bytes when production-secure cookies are enabled (got %d)", MinCookieKeyBytes, len(key))
	}
	if key == CookieKeyDefaultSentinel || key == CookieKeyREADMESentinel {
		return fmt.Errorf("COOKIE_KEY must not use the default/example sentinel value when production-secure cookies are enabled")
	}
	return nil
}
