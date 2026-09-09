package recognize

import "testing"

func TestProportionsScore_LengthRatio(t *testing.T) {
	tmpl := NormalizeResult{Strokes: []NormStroke{
		buildNormStroke([]Point{{X: 0.2, Y: 0.2}, {X: 0.8, Y: 0.2}}),
		buildNormStroke([]Point{{X: 0.3, Y: 0.4}, {X: 0.7, Y: 0.4}, {X: 0.5, Y: 0.8}}),
	}}
	// Same starts/centroids roughly, but stroke 2 path made much longer via detour.
	learner := NormalizeResult{Strokes: []NormStroke{
		buildNormStroke([]Point{{X: 0.2, Y: 0.2}, {X: 0.8, Y: 0.2}}),
		buildNormStroke([]Point{
			{X: 0.3, Y: 0.4}, {X: 0.9, Y: 0.1}, {X: 0.3, Y: 0.4},
			{X: 0.7, Y: 0.4}, {X: 0.5, Y: 0.8},
		}),
	}}
	got := proportionsScore(learner.Strokes, tmpl.Strokes)
	matched := proportionsScore(tmpl.Strokes, tmpl.Strokes)
	if matched < 0.95 {
		t.Fatalf("matched proportions want ~1, got %.3f", matched)
	}
	if got > 0.75 {
		t.Fatalf("distorted proportions should be weak, got %.3f", got)
	}
	t.Logf("matched=%.3f distorted=%.3f", matched, got)
}
