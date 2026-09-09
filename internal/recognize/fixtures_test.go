package recognize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type assessmentFixture struct {
	Name                 string   `json:"name"`
	Target               string   `json:"target"`
	Strokes              []Stroke `json:"strokes"`
	ExpectPass           bool     `json:"expectPass"`
	ExpectFeedbackCode   string   `json:"expectFeedbackCode"`
	ExpectWeakCriterion  string   `json:"expectWeakCriterion"`
}

func loadFixtureFile(t *testing.T, path string) assessmentFixture {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var f assessmentFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("json %s: %v", path, err)
	}
	return f
}

func walkFixtures(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return paths
}

func TestHiragana5FixturePacks(t *testing.T) {
	r := loadTemplates(t)
	root := filepath.Join("testdata", "hiragana5")
	paths := walkFixtures(t, root)
	if len(paths) == 0 {
		t.Fatal("no fixtures found")
	}
	for _, path := range paths {
		path := path
		rel, _ := filepath.Rel(root, path)
		t.Run(rel, func(t *testing.T) {
			f := loadFixtureFile(t, path)
			a, err := r.Assess(f.Target, f.Strokes, 300, 300)
			if err != nil {
				t.Fatalf("assess: %v", err)
			}
			if a.Pass != f.ExpectPass {
				t.Fatalf("pass=%t want=%t score=%.3f diag=%+v feedback=%#v reasons=%v",
					a.Pass, f.ExpectPass, a.Score, a.Diagnostics, a.Feedback, a.Reasons)
			}
			if f.ExpectFeedbackCode == "" {
				return
			}
			if len(a.Feedback) == 0 {
				t.Fatalf("expected feedback code %s, got none (diag=%+v)", f.ExpectFeedbackCode, a.Diagnostics)
			}
			primary := a.Feedback[0].Code
			ok := primary == f.ExpectFeedbackCode
			if !ok && len(a.Feedback) > 1 && a.Feedback[1].Code == f.ExpectFeedbackCode {
				ok = true
			}
			if !ok {
				t.Fatalf("expected feedback code %s in top-2, got %#v diag=%+v",
					f.ExpectFeedbackCode, a.Feedback, a.Diagnostics)
			}
			if f.ExpectFeedbackCode == CodeProportions && a.Diagnostics != nil {
				t.Logf("[FIX] proportions fixture=%s pass=%t prop=%.3f place=%.3f shape=%.3f feedback=%v",
					rel, a.Pass, a.Diagnostics.Proportions, a.Diagnostics.RelativePlacement, a.Diagnostics.Shape, feedbackCodes(a.Feedback))
			}
			if f.ExpectWeakCriterion != "" && a.Diagnostics != nil {
				weak := criterionByCode(a.Diagnostics, f.ExpectWeakCriterion)
				if weak >= SoftCriterionThreshold {
					t.Fatalf("expectWeakCriterion %s score=%.3f want < %.2f", f.ExpectWeakCriterion, weak, SoftCriterionThreshold)
				}
			}
			for _, item := range a.Feedback {
				if item.Message == "" {
					t.Fatalf("empty message for code=%s", item.Code)
				}
			}
			t.Logf("pass=%t score=%.3f feedback=%v", a.Pass, a.Score, feedbackCodes(a.Feedback))
		})
	}
}

func criterionByCode(d *Diagnostics, code string) float64 {
	switch code {
	case "stroke_count":
		return d.StrokeCount
	case CodeStrokeOrder:
		return d.StrokeOrder
	case "start_end_direction":
		return d.StartEndDirection
	case CodeRelativePlacement:
		return d.RelativePlacement
	case CodeProportions:
		return d.Proportions
	case CodeShape:
		return d.Shape
	default:
		return 1
	}
}

func feedbackCodes(items []FeedbackItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Code
	}
	return out
}
