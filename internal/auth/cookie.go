package auth

import (
	"fmt"
	"strings"
)

// Forbidden production cookie-key sentinels (defaults / README / compose examples).
const (
	CookieKeyDefaultSentinel    = "change-me-please-32-bytes-min"
	CookieKeyREADMESentinel     = "please-change-this-32-bytes-min"
	CookieKeyComposeSentinel    = "replace-me-with-a-long-random-cookie-key"
	CookieKeyEnvExampleSentinel = "your-secure-random-cookie-key-here"
	CookieKeyDevComposeSentinel = "dev-secret-key-change-me-32bytes!!"
	MinCookieKeyBytes           = 32
)

// cookieKeySentinels are documented placeholders that must never be used when
// production-secure cookies are enabled.
var cookieKeySentinels = []string{
	CookieKeyDefaultSentinel,
	CookieKeyREADMESentinel,
	CookieKeyComposeSentinel,
	CookieKeyEnvExampleSentinel,
	CookieKeyDevComposeSentinel,
}

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

func isCookieKeySentinel(key string) bool {
	for _, s := range cookieKeySentinels {
		if key == s {
			return true
		}
	}
	return false
}

// IsWeakCookieKey reports empty, short, or known sentinel keys.
func IsWeakCookieKey(key string) bool {
	if key == "" || len(key) < MinCookieKeyBytes {
		return true
	}
	return isCookieKeySentinel(key)
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
	if isCookieKeySentinel(key) {
		return fmt.Errorf("COOKIE_KEY must not use the default/example sentinel value when production-secure cookies are enabled")
	}
	return nil
}
