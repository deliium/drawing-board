package ids

import (
	"errors"
	"strings"
	"testing"
)

func TestNewReturnsCanonicalUUID(t *testing.T) {
	id := New()
	if !Valid(id) {
		t.Fatalf("New() not valid: %q", id)
	}
	if len(id) != 36 {
		t.Fatalf("len=%d want 36", len(id))
	}
	if id != strings.ToLower(id) {
		t.Fatalf("expected lowercase: %q", id)
	}
	if id != strings.ReplaceAll(id, " ", "") {
		t.Fatalf("unexpected spaces: %q", id)
	}
}

func TestParseAcceptsCanonicalAndNormalizes(t *testing.T) {
	raw := "550E8400-E29B-41D4-A716-446655440000"
	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("got %q", got)
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, in := range []string{"", "   ", "not-a-uuid", "123", "550e8400e29b41d4a716446655440000zzzz"} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("Parse(%q) err=%v want ErrInvalidID", in, err)
		}
		if Valid(in) {
			t.Fatalf("Valid(%q) true want false", in)
		}
	}
}

func TestErrorCode(t *testing.T) {
	if ErrorCode(ErrInvalidID) != "invalid_input" {
		t.Fatalf("ErrorCode=%s", ErrorCode(ErrInvalidID))
	}
}
