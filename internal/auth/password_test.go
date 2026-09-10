package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyBcrypt(t *testing.T) {
	hash, err := HashPassword("password1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt prefix, got %q", hash[:min(8, len(hash))])
	}
	ok, err := VerifyPassword(hash, "password1")
	if err != nil || !ok {
		t.Fatalf("verify bcrypt: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(hash, "wrongpass")
	if err != nil || ok {
		t.Fatalf("wrong password should fail: ok=%v err=%v", ok, err)
	}
}

func TestVerifyPasswordRejectsNonBcrypt(t *testing.T) {
	// Pre-migration unsalted SHA-256 hex of "password1" must not authenticate.
	legacy := "0b14d501a594442a01c6859541bcb3e8164d183d32937b851835442f69d5c94e"
	ok, err := VerifyPassword(legacy, "password1")
	if err != nil || ok {
		t.Fatalf("legacy hash must not verify: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("", "password1")
	if err != nil || ok {
		t.Fatalf("empty hash must not verify: ok=%v err=%v", ok, err)
	}
}

func TestProductionSecureMode(t *testing.T) {
	cases := []struct {
		env, secure string
		want        bool
	}{
		{"production", "", true},
		{"Production", "", true},
		{"", "true", true},
		{"", "1", true},
		{"", "yes", true},
		{"dev", "false", false},
		{"", "", false},
	}
	for _, tc := range cases {
		if got := ProductionSecureMode(tc.env, tc.secure); got != tc.want {
			t.Fatalf("ProductionSecureMode(%q,%q)=%v want %v", tc.env, tc.secure, got, tc.want)
		}
	}
}

func TestValidateCookieKey(t *testing.T) {
	strong := "a-sufficiently-long-random-cookie-key!!"
	if err := ValidateCookieKey(strong, true); err != nil {
		t.Fatalf("strong key should pass: %v", err)
	}
	sentinels := []string{
		CookieKeyDefaultSentinel,
		CookieKeyREADMESentinel,
		CookieKeyComposeSentinel,
		CookieKeyEnvExampleSentinel,
		CookieKeyDevComposeSentinel,
	}
	for _, sentinel := range sentinels {
		if err := ValidateCookieKey(sentinel, true); err == nil {
			t.Fatalf("sentinel %q must fail in secure mode", sentinel)
		}
		if !IsWeakCookieKey(sentinel) {
			t.Fatalf("IsWeakCookieKey should detect sentinel %q", sentinel)
		}
		if err := ValidateCookieKey(sentinel, false); err != nil {
			t.Fatalf("non-secure mode should allow weak key %q: %v", sentinel, err)
		}
	}
	if err := ValidateCookieKey("short", true); err == nil {
		t.Fatal("short key must fail in secure mode")
	}
	if !IsWeakCookieKey("x") {
		t.Fatal("IsWeakCookieKey should detect short keys")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
