// Package ids provides UUID generation and validation for surrogate entity keys.
package ids

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
)

// ErrInvalidID is returned when a value is not a canonical RFC 4122 UUID string.
var ErrInvalidID = errors.New("invalid_id")

// New returns a new random UUID v4 in canonical lowercase hyphenated form.
func New() string {
	id := uuid.NewString()
	idsLog("DEBUG", "[ids.New] generated")
	return id
}

// Valid reports whether s is a parseable UUID (any version/variant accepted by google/uuid).
func Valid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// Parse validates and returns the canonical UUID string, or ErrInvalidID.
func Parse(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		idsLog("DEBUG", "[ids.Parse] reject reason=empty")
		return "", ErrInvalidID
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		idsLog("DEBUG", "[ids.Parse] reject reason=malformed")
		return "", ErrInvalidID
	}
	// Canonical lowercase hyphenated form (google/uuid String()).
	out := parsed.String()
	return out, nil
}

// ErrorCode maps ids errors to stable API/WS codes.
func ErrorCode(err error) string {
	if errors.Is(err, ErrInvalidID) {
		return "invalid_input"
	}
	return "invalid_input"
}

func idsLog(level, msg string) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "info", "warn", "error":
		if level == "DEBUG" {
			return
		}
	}
	log.Printf("%s %s", level, msg)
}
