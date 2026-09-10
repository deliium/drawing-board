package docguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Phrases that oversell recognition as AI/ML or claim an active ONNX upgrade path.
// Honest disclaimers ("not a trained AI model", "not calibrated confidence") and
// historical "removed" notes are allowed; active capability claims are not.
var forbiddenREADMEPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{name: "AI-powered", re: regexp.MustCompile(`(?i)AI-powered`)},
	{name: "AI Recognition", re: regexp.MustCompile(`(?i)AI\s+Recognition`)},
	{name: "confidence scores", re: regexp.MustCompile(`(?i)confidence\s+scores`)},
	{name: "calibrated AI confidence claim", re: regexp.MustCompile(`(?i)calibrated\s+AI\s+confidence`)},
	{name: "X% accurate handwriting claim", re: regexp.MustCompile(`(?i)\d+%\s+accurate\s+handwriting`)},
	{name: "Machine learning-based", re: regexp.MustCompile(`(?i)Machine\s+learning-based`)},
	{name: "Higher accuracy (ONNX overclaim)", re: regexp.MustCompile(`(?i)Higher\s+accuracy`)},
	{name: "ONNX model for advanced recognition", re: regexp.MustCompile(`(?i)ONNX\s+model\s+for\s+advanced\s+recognition`)},
	{name: "ONNX Recognizer (Optional)", re: regexp.MustCompile(`(?i)ONNX\s+Recognizer\s*\(\s*Optional`)},
	{name: "make onnx-model as setup", re: regexp.MustCompile(`(?i)make\s+onnx-model`)},
	{name: "ONNX_MODEL env capability", re: regexp.MustCompile(`(?i)ONNX_MODEL\s*=`)},
	{name: "AI-authored curriculum", re: regexp.MustCompile(`(?i)AI[- ]authored\s+curriculum`)},
	{name: "expert-reviewed curriculum overclaim", re: regexp.MustCompile(`(?i)expert-reviewed\s+curriculum`)},
	{name: "AI lesson marketing", re: regexp.MustCompile(`(?i)AI\s+lesson`)},
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
			"Recognition must not claim AI/ML confidence or an active ONNX upgrade path.",
			strings.Join(hits, "\n  - "))
	}
}

// When pack LICENSES.md marks strokes.json / traces.json as CC BY-SA, README must not
// claim that curriculum stroke/trace geometry is under CC0, and must surface the share-alike license.
var readmeStrokeTraceCC0Claim = regexp.MustCompile(`(?i)stroke/?trace[^\n.]{0,120}\bunder\s+CC0`)

func TestREADMEStrokeTraceLicenseMatchesPack(t *testing.T) {
	root := moduleRoot(t)
	licPath := filepath.Join(root, "content", "hiragana5", "LICENSES.md")
	licRaw, err := os.ReadFile(licPath)
	if err != nil {
		t.Fatalf("read LICENSES.md: %v", err)
	}
	packShareAlike := false
	for _, line := range strings.Split(string(licRaw), "\n") {
		if !strings.Contains(line, "CC BY-SA") {
			continue
		}
		if strings.Contains(line, "strokes.json") || strings.Contains(line, "traces.json") {
			packShareAlike = true
			break
		}
	}
	if !packShareAlike {
		t.Skip("hiragana5 LICENSES.md does not mark strokes.json/traces.json as CC BY-SA")
	}

	readmePath := filepath.Join(root, "README.md")
	readmeRaw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(readmeRaw)
	if loc := readmeStrokeTraceCC0Claim.FindStringIndex(readme); loc != nil {
		line := 1 + strings.Count(readme[:loc[0]], "\n")
		end := loc[1] + 40
		if end > len(readme) {
			end = len(readme)
		}
		snippet := strings.TrimSpace(strings.ReplaceAll(readme[loc[0]:end], "\n", " "))
		t.Fatalf("README.md claims stroke/trace geometry under CC0 while content/hiragana5/LICENSES.md marks those files CC BY-SA @ line %d: %s",
			line, snippet)
	}
	if !strings.Contains(readme, "CC BY-SA") {
		t.Fatal("README.md must mention CC BY-SA when hiragana5 stroke/trace geometry is share-alike licensed (see content/hiragana5/LICENSES.md)")
	}
}
