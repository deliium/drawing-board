package recognize

import (
	"path/filepath"
	"strings"
	"testing"
)

// Eval summary for hiragana5 fixtures (engineering criteria, not learner-facing accuracy %).
// T_pass = PassThreshold (0.70). Run with: go test ./internal/recognize -run Eval -v
func TestEvalHiragana5Summary(t *testing.T) {
	r, err := NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	goldPass, goldFail := 0, 0
	wrongPass, wrongReject := 0, 0
	suitePass := map[string]int{}
	suiteFail := map[string]int{}
	suiteUnexpectedPass := map[string]int{}
	suiteMissCode := map[string]int{}

	root := filepath.Join("testdata", "hiragana5")
	for _, path := range walkFixtures(t, root) {
		f := loadFixtureFile(t, path)
		rel, _ := filepath.Rel(root, path)
		suite := suiteName(rel)
		a, err := r.Assess(f.Target, f.Strokes, 300, 300)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if a.Pass {
			suitePass[suite]++
		} else {
			suiteFail[suite]++
		}
		if f.ExpectPass {
			if a.Pass && a.Score >= PassThreshold {
				goldPass++
			} else {
				goldFail++
				t.Logf("gold_fail path=%s pass=%t score=%.3f", rel, a.Pass, a.Score)
			}
		} else {
			if a.Pass {
				wrongPass++
				suiteUnexpectedPass[suite]++
				t.Logf("incorrect_unexpected_pass path=%s score=%.3f", rel, a.Score)
			} else {
				wrongReject++
			}
			if f.ExpectFeedbackCode != "" {
				ok := false
				for _, item := range a.Feedback {
					if item.Code == f.ExpectFeedbackCode {
						ok = true
						break
					}
				}
				if !ok {
					suiteMissCode[suite]++
					t.Logf("incorrect_miss_code path=%s want=%s got=%v", rel, f.ExpectFeedbackCode, feedbackCodes(a.Feedback))
				}
			}
		}
	}

	t.Logf("eval set=%s version=%s T_pass=%.2f gold_pass=%d gold_fail=%d incorrect_reject=%d incorrect_unexpected_pass=%d",
		r.SetID(), r.Version(), PassThreshold, goldPass, goldFail, wrongReject, wrongPass)
	for _, suite := range []string{"gold", "incorrect/count", "incorrect/order", "incorrect/direction", "incorrect/placement", "incorrect/proportions", "incorrect/wrong_glyph", "short_strokes"} {
		t.Logf("suite=%s pass=%d fail=%d unexpected_pass=%d miss_code=%d",
			suite, suitePass[suite], suiteFail[suite], suiteUnexpectedPass[suite], suiteMissCode[suite])
	}

	if goldFail != 0 {
		t.Fatalf("gold failures: %d (all gold fixtures must pass T_pass)", goldFail)
	}
	if wrongPass != 0 {
		t.Fatalf("incorrect unexpected passes: %d", wrongPass)
	}
	missTotal := 0
	for _, n := range suiteMissCode {
		missTotal += n
	}
	if missTotal != 0 {
		t.Fatalf("incorrect fixtures missing expected feedback code: %d", missTotal)
	}
}

func suiteName(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return rel
	}
	if parts[0] == "gold" || parts[0] == "short_strokes" {
		return parts[0]
	}
	if parts[0] == "incorrect" && len(parts) >= 2 {
		return "incorrect/" + parts[1]
	}
	return parts[0]
}
