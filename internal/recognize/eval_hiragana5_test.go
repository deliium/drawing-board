package recognize

import "testing"

// Eval summary for hiragana5 fixtures (engineering criteria, not learner-facing accuracy %).
// T_pass = PassThreshold (0.70). Run with: go test ./internal/recognize -run Eval -v
func TestEvalHiragana5Summary(t *testing.T) {
	r, err := NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	goldPass, goldFail := 0, 0
	wrongPass, wrongReject := 0, 0

	for _, glyph := range r.order {
		strokes := make([]Stroke, len(r.templates[glyph]))
		for i, s := range r.templates[glyph] {
			strokes[i] = Stroke{Points: append([]Point(nil), s.Points...)}
		}
		a, err := r.Assess(glyph, strokes, 300, 300)
		if err != nil {
			t.Fatalf("gold %s: %v", glyph, err)
		}
		if a.Pass && a.Score >= PassThreshold {
			goldPass++
		} else {
			goldFail++
		}
		t.Logf("gold glyph=%s pass=%t score=%.3f", glyph, a.Pass, a.Score)
	}

	for _, target := range r.order {
		for _, other := range r.order {
			if other == target {
				continue
			}
			strokes := make([]Stroke, len(r.templates[other]))
			for i, s := range r.templates[other] {
				strokes[i] = Stroke{Points: append([]Point(nil), s.Points...)}
			}
			a, err := r.Assess(target, strokes, 300, 300)
			if err != nil {
				t.Fatalf("wrong %s/%s: %v", target, other, err)
			}
			if a.Pass {
				wrongPass++
			} else {
				wrongReject++
			}
		}
	}

	t.Logf("eval set=%s version=%s T_pass=%.2f gold_pass=%d gold_fail=%d wrong_pass=%d wrong_reject=%d",
		r.SetID(), r.Version(), PassThreshold, goldPass, goldFail, wrongPass, wrongReject)

	if goldFail != 0 {
		t.Fatalf("gold failures: %d (all gold fixtures must pass T_pass)", goldFail)
	}
	if wrongPass != 0 {
		t.Fatalf("wrong-character passes: %d (must not pass requested target)", wrongPass)
	}
}
