package recognize

import "testing"

func TestNormalize_TranslationScaleInvariance(t *testing.T) {
	base := []Stroke{
		{Points: []Point{{X: 0.2, Y: 0.2}, {X: 0.8, Y: 0.2}}},
		{Points: []Point{{X: 0.5, Y: 0.1}, {X: 0.5, Y: 0.9}}},
	}
	a := normalizeInk(base)
	b := normalizeInk(scaleStrokes(base, 200, 200, 40, 30))
	if a.Empty || b.Empty {
		t.Fatalf("unexpected empty: a=%v b=%v", a.Empty, b.Empty)
	}
	if len(a.Strokes) != len(b.Strokes) {
		t.Fatalf("stroke count %d vs %d", len(a.Strokes), len(b.Strokes))
	}
	for i := range a.Strokes {
		da := a.Strokes[i].Centroid
		db := b.Strokes[i].Centroid
		if abs(da.X-db.X) > 1e-6 || abs(da.Y-db.Y) > 1e-6 {
			t.Fatalf("centroid drift stroke %d: %+v vs %+v", i, da, db)
		}
	}
}

func TestNormalize_DegenerateVertical(t *testing.T) {
	strokes := []Stroke{{Points: []Point{{X: 0.5, Y: 0.1}, {X: 0.5, Y: 0.9}}}}
	res := normalizeInk(strokes)
	if res.Empty || !res.DegenerateAxis {
		t.Fatalf("want degenerateAxis: empty=%t deg=%t", res.Empty, res.DegenerateAxis)
	}
	if res.NormalStrokeCount != 1 {
		t.Fatalf("want 1 normal, short=%d normal=%d", res.ShortStrokeCount, res.NormalStrokeCount)
	}
}

func TestNormalize_OnePointShort(t *testing.T) {
	strokes := []Stroke{{Points: []Point{{X: 0.4, Y: 0.5}}}}
	res := normalizeInk(strokes)
	if res.Empty || res.ShortStrokeCount != 1 || res.NormalStrokeCount != 0 {
		t.Fatalf("want short: %+v", res)
	}
	if res.Strokes[0].Kind != StrokeShort {
		t.Fatalf("kind=%v", res.Strokes[0].Kind)
	}
}

func TestNormalize_TinyPathShort(t *testing.T) {
	// Short marks stay short relative to a longer stroke in the same ink bbox.
	strokes := []Stroke{
		{Points: []Point{{X: 0.1, Y: 0.1}, {X: 0.9, Y: 0.9}}},
		{Points: []Point{{X: 0.5, Y: 0.5}, {X: 0.5 + ShortStrokeLength/4, Y: 0.5}}},
	}
	res := normalizeInk(strokes)
	if res.ShortStrokeCount != 1 || res.NormalStrokeCount != 1 {
		t.Fatalf("expected 1 short + 1 normal, short=%d normal=%d lens=[%.4f %.4f]",
			res.ShortStrokeCount, res.NormalStrokeCount, res.Strokes[0].Length, res.Strokes[1].Length)
	}
}

func TestNormalize_DropsEmptyPointStrokes(t *testing.T) {
	strokes := []Stroke{
		{Points: nil},
		{Points: []Point{{X: 0.1, Y: 0.1}, {X: 0.9, Y: 0.9}}},
	}
	res := normalizeInk(strokes)
	if res.DroppedEmpty != 1 || len(res.Strokes) != 1 {
		t.Fatalf("dropped=%d strokes=%d", res.DroppedEmpty, len(res.Strokes))
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
