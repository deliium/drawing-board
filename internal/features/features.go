// Package features parses env-driven learning product flags used as kill switches
// for practice/progress/review/audio surfaces (migrations stay forward-only).
package features

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Flags are process-start learning surface kill switches.
type Flags struct {
	Practice bool `json:"practice"`
	Progress bool `json:"progress"`
	Review   bool `json:"review"`
	Audio    bool `json:"audio"`
}

// Env keys (documented in README).
const (
	EnvPractice = "FEATURE_PRACTICE"
	EnvProgress = "FEATURE_PROGRESS"
	EnvReview   = "FEATURE_REVIEW"
	EnvAudio    = "FEATURE_AUDIO"
)

// ParseFromEnv reads FEATURE_* from the process environment.
// Empty values default to true (dev-friendly); operators set 0 explicitly for prod cutovers.
func ParseFromEnv() (Flags, error) {
	return Parse(os.Getenv)
}

// Parse reads FEATURE_* via getenv. Empty → true. Invalid REVIEW without PROGRESS → error.
func Parse(getenv func(string) string) (Flags, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	rawPractice := getenv(EnvPractice)
	rawProgress := getenv(EnvProgress)
	rawReview := getenv(EnvReview)
	rawAudio := getenv(EnvAudio)
	if debugLogs() {
		log.Printf("DEBUG [features.Parse] raw %s=%q %s=%q %s=%q %s=%q",
			EnvPractice, rawPractice, EnvProgress, rawProgress, EnvReview, rawReview, EnvAudio, rawAudio)
	}

	practice, err := parseBool(rawPractice, true)
	if err != nil {
		return Flags{}, fmt.Errorf("%s: %w", EnvPractice, err)
	}
	progress, err := parseBool(rawProgress, true)
	if err != nil {
		return Flags{}, fmt.Errorf("%s: %w", EnvProgress, err)
	}
	review, err := parseBool(rawReview, true)
	if err != nil {
		return Flags{}, fmt.Errorf("%s: %w", EnvReview, err)
	}
	audio, err := parseBool(rawAudio, true)
	if err != nil {
		return Flags{}, fmt.Errorf("%s: %w", EnvAudio, err)
	}

	f := Flags{Practice: practice, Progress: progress, Review: review, Audio: audio}
	if f.Review && !f.Progress {
		return Flags{}, fmt.Errorf("invalid feature combo: %s=1 requires %s=1", EnvReview, EnvProgress)
	}
	if debugLogs() {
		log.Printf("DEBUG [features.Parse] resolved practice=%v progress=%v review=%v audio=%v",
			f.Practice, f.Progress, f.Review, f.Audio)
	}
	return f, nil
}

func debugLogs() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug", "trace", "verbose":
		return true
	default:
		return false
	}
}

func parseBool(raw string, def bool) (bool, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return def, nil
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool %q (want 0|1|true|false)", raw)
	}
}

// LogStartup writes the cutover verification line (INFO).
func LogStartup(f Flags, schemaVersion int) {
	log.Printf("INFO [main] features practice=%v progress=%v review=%v audio=%v schema_version=%d",
		f.Practice, f.Progress, f.Review, f.Audio, schemaVersion)
}
