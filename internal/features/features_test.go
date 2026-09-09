package features

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestParseDefaultsOn(t *testing.T) {
	f, err := Parse(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if !f.Practice || !f.Progress || !f.Review || !f.Audio {
		t.Fatalf("expected all on by default, got %+v", f)
	}
}

func TestParseInvalidReviewWithoutProgress(t *testing.T) {
	_, err := Parse(func(k string) string {
		switch k {
		case EnvProgress:
			return "0"
		case EnvReview:
			return "1"
		default:
			return "1"
		}
	})
	if err == nil {
		t.Fatal("expected invalid combo error")
	}
	if !strings.Contains(err.Error(), EnvReview) {
		t.Fatalf("error should name review: %v", err)
	}
	t.Logf("invalid combo: %v", err)
}

func TestParseExplicitOff(t *testing.T) {
	f, err := Parse(func(k string) string {
		return "0"
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Practice || f.Progress || f.Review || f.Audio {
		t.Fatalf("expected all off, got %+v", f)
	}
}

func TestParseBadValue(t *testing.T) {
	_, err := Parse(func(k string) string {
		if k == EnvPractice {
			return "maybe"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLogStartup(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	LogStartup(Flags{Practice: true, Progress: false, Review: false, Audio: true}, 4)
	out := buf.String()
	if !strings.Contains(out, "INFO [main] features practice=true progress=false review=false audio=true schema_version=4") {
		t.Fatalf("unexpected startup log: %q", out)
	}
}
