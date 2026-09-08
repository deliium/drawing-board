package recognize

import (
	"errors"
	"testing"
)

func loadTemplates(t *testing.T) *TargetCompareRecognizer {
	t.Helper()
	r, err := NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("NewTargetCompareRecognizer: %v", err)
	}
	return r
}

func templateStrokes(r *TargetCompareRecognizer, glyph string) []Stroke {
	src := r.templates[glyph]
	out := make([]Stroke, len(src))
	for i, s := range src {
		pts := make([]Point, len(s.Points))
		copy(pts, s.Points)
		out[i] = Stroke{Points: pts}
	}
	return out
}

func scaleStrokes(strokes []Stroke, sx, sy, tx, ty float64) []Stroke {
	out := make([]Stroke, len(strokes))
	for i, s := range strokes {
		pts := make([]Point, len(s.Points))
		for j, p := range s.Points {
			pts[j] = Point{X: p.X*sx + tx, Y: p.Y*sy + ty}
		}
		out[i] = Stroke{Points: pts}
	}
	return out
}

func TestTargetCompare_GoldFixturesPass(t *testing.T) {
	r := loadTemplates(t)
	for _, glyph := range r.order {
		strokes := templateStrokes(r, glyph)
		a, err := r.Assess(glyph, strokes, 300, 300)
		if err != nil {
			t.Fatalf("%s: assess: %v", glyph, err)
		}
		if !a.Pass {
			t.Fatalf("%s: expected pass, score=%.3f reasons=%v", glyph, a.Score, a.Reasons)
		}
		if a.Score < PassThreshold {
			t.Fatalf("%s: score %.3f < T_pass %.2f", glyph, a.Score, PassThreshold)
		}
		if len(a.Candidates) == 0 || a.Candidates[0].Text != glyph {
			t.Fatalf("%s: top candidate want %s got %#v", glyph, glyph, a.Candidates)
		}
		if a.ScoreKind != ScoreKindMatch {
			t.Fatalf("%s: scoreKind=%q", glyph, a.ScoreKind)
		}
	}
}

func TestTargetCompare_WrongCharacterDoesNotPass(t *testing.T) {
	r := loadTemplates(t)
	for _, target := range r.order {
		for _, other := range r.order {
			if other == target {
				continue
			}
			strokes := templateStrokes(r, other)
			a, err := r.Assess(target, strokes, 300, 300)
			if err != nil {
				t.Fatalf("assess %s with %s strokes: %v", target, other, err)
			}
			if a.Pass {
				t.Fatalf("target=%s strokes=%s unexpectedly passed score=%.3f", target, other, a.Score)
			}
		}
	}
}

func TestTargetCompare_NearMissStrokeCount(t *testing.T) {
	r := loadTemplates(t)
	strokes := templateStrokes(r, "あ")
	// Drop last stroke → near-miss / non-perfect.
	truncated := strokes[:len(strokes)-1]
	a, err := r.Assess("あ", truncated, 300, 300)
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if a.Pass && a.Score >= 0.95 {
		t.Fatalf("near-miss should not look perfect: pass=%t score=%.3f", a.Pass, a.Score)
	}
	found := false
	for _, reason := range a.Reasons {
		if reason == "stroke_count_mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stroke_count_mismatch in reasons: %v", a.Reasons)
	}
}

func TestTargetCompare_EmptyStrokes(t *testing.T) {
	r := loadTemplates(t)
	a, err := r.Assess("い", nil, 300, 300)
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if a.Pass || a.Score != 0 {
		t.Fatalf("empty: pass=%t score=%f", a.Pass, a.Score)
	}
}

func TestTargetCompare_UnsupportedTarget(t *testing.T) {
	r := loadTemplates(t)
	_, err := r.Assess("漢", templateStrokes(r, "あ"), 300, 300)
	if !errors.Is(err, ErrUnsupportedTarget) {
		t.Fatalf("want ErrUnsupportedTarget, got %v", err)
	}
}

func TestTargetCompare_NormalizationStability(t *testing.T) {
	r := loadTemplates(t)
	base := templateStrokes(r, "う")
	scaled := scaleStrokes(base, 200, 200, 40, 30)
	a1, err := r.Assess("う", base, 300, 300)
	if err != nil {
		t.Fatalf("base: %v", err)
	}
	a2, err := r.Assess("う", scaled, 300, 300)
	if err != nil {
		t.Fatalf("scaled: %v", err)
	}
	if !a1.Pass || !a2.Pass {
		t.Fatalf("both should pass: base=%v scaled=%v", a1, a2)
	}
	diff := a1.Score - a2.Score
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.05 {
		t.Fatalf("normalization unstable: base=%.3f scaled=%.3f", a1.Score, a2.Score)
	}
}

func TestTargetCompare_RecognizeSetsScoreKind(t *testing.T) {
	r := loadTemplates(t)
	cands, err := r.Recognize([]Stroke{{Points: []Point{{X: 10, Y: 10}, {X: 50, Y: 10}}}}, 300, 300, 5)
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if len(cands) == 0 {
		t.Fatal("expected candidates")
	}
	for _, c := range cands {
		if c.ScoreKind != ScoreKindMatch {
			t.Fatalf("scoreKind=%q want match", c.ScoreKind)
		}
	}
}
