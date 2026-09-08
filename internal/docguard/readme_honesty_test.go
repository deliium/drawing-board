package docguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Phrases that oversell the current heuristic recognizer or unimplemented ONNX path.
// Honest disclaimers such as "not a trained AI model" / "not calibrated confidence" are allowed.
var forbiddenREADMEPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{name: "AI-powered", re: regexp.MustCompile(`(?i)AI-powered`)},
	{name: "AI Recognition", re: regexp.MustCompile(`(?i)AI\s+Recognition`)},
	{name: "confidence scores", re: regexp.MustCompile(`(?i)confidence\s+scores`)},
	{name: "Machine learning-based", re: regexp.MustCompile(`(?i)Machine\s+learning-based`)},
	{name: "Higher accuracy (ONNX overclaim)", re: regexp.MustCompile(`(?i)Higher\s+accuracy`)},
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getcwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking from test cwd")
		}
		dir = parent
	}
}

func TestREADMEDoesNotMarketHeuristicAsAI(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(raw)

	var hits []string
	for _, p := range forbiddenREADMEPatterns {
		if loc := p.re.FindStringIndex(text); loc != nil {
			line := 1 + strings.Count(text[:loc[0]], "\n")
			end := loc[1] + 40
			if end > len(text) {
				end = len(text)
			}
			snippet := strings.TrimSpace(text[loc[0]:end])
			snippet = strings.ReplaceAll(snippet, "\n", " ")
			hits = append(hits, p.name+" @ line "+strconv.Itoa(line)+": "+snippet)
		}
	}
	if len(hits) > 0 {
		t.Fatalf("README.md reintroduced dishonest recognition marketing:\n  - %s\n"+
			"Heuristic recognizer must not be labeled AI; scores are not calibrated confidence.",
			strings.Join(hits, "\n  - "))
	}
}
