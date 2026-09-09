package recognize

import "math"

// StrokeKind classifies a stroke after global normalization.
type StrokeKind int

const (
	StrokeNormal StrokeKind = iota
	StrokeShort             // path length ≤ L_short or ≤1 distinct point (dot / tap)
)

// NormStroke is one stroke in shared unit space with derived features.
type NormStroke struct {
	Points   []Point
	Kind     StrokeKind
	Length   float64
	Centroid Point
	Start    Point
	End      Point
}

// NormalizeResult is the shared learner/canonical normalization pipeline output.
type NormalizeResult struct {
	Strokes           []NormStroke
	Empty             bool
	DroppedEmpty      int
	DegenerateAxis    bool
	ShortStrokeCount  int
	NormalStrokeCount int
}

// normalizeInk applies the documented normalization contract (D2):
// drop empty-point strokes, whole-ink AABB → origin, scale by 1/max(w,h), classify short vs normal.
func normalizeInk(strokes []Stroke) NormalizeResult {
	kept := make([]Stroke, 0, len(strokes))
	dropped := 0
	for _, s := range strokes {
		if len(s.Points) == 0 {
			dropped++
			continue
		}
		kept = append(kept, Stroke{Points: append([]Point(nil), s.Points...)})
	}
	if len(kept) == 0 {
		logf("DEBUG", "[recognize.normalize] empty=true droppedEmpty=%d", dropped)
		return NormalizeResult{Empty: true, DroppedEmpty: dropped}
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, s := range kept {
		for _, p := range s.Points {
			if p.X < minX {
				minX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
	}
	w := maxX - minX
	h := maxY - minY
	degenerate := false
	if w < EpsilonBounds {
		w = EpsilonBounds
		degenerate = true
	}
	if h < EpsilonBounds {
		h = EpsilonBounds
		degenerate = true
	}
	scale := 1 / math.Max(w, h)

	out := make([]NormStroke, len(kept))
	shortN, normalN := 0, 0
	for i, s := range kept {
		pts := make([]Point, len(s.Points))
		for j, p := range s.Points {
			pts[j] = Point{X: (p.X - minX) * scale, Y: (p.Y - minY) * scale}
		}
		ns := buildNormStroke(pts)
		out[i] = ns
		if ns.Kind == StrokeShort {
			shortN++
		} else {
			normalN++
		}
	}
	logf("DEBUG", "[recognize.normalize] strokes=%d droppedEmpty=%d short=%d normal=%d degenerateAxis=%t",
		len(out), dropped, shortN, normalN, degenerate)
	return NormalizeResult{
		Strokes:           out,
		DroppedEmpty:      dropped,
		DegenerateAxis:    degenerate,
		ShortStrokeCount:  shortN,
		NormalStrokeCount: normalN,
	}
}

func buildNormStroke(pts []Point) NormStroke {
	ns := NormStroke{
		Points: pts,
		Start:  pts[0],
		End:    pts[len(pts)-1],
	}
	ns.Length = pathLength(pts)
	ns.Centroid = centroid(pts)
	if distinctPointCount(pts) <= 1 || ns.Length <= ShortStrokeLength {
		ns.Kind = StrokeShort
	} else {
		ns.Kind = StrokeNormal
	}
	return ns
}

func pathLength(pts []Point) float64 {
	if len(pts) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(pts); i++ {
		dx := pts[i].X - pts[i-1].X
		dy := pts[i].Y - pts[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

func centroid(pts []Point) Point {
	if len(pts) == 0 {
		return Point{}
	}
	var sx, sy float64
	for _, p := range pts {
		sx += p.X
		sy += p.Y
	}
	n := float64(len(pts))
	return Point{X: sx / n, Y: sy / n}
}

func distinctPointCount(pts []Point) int {
	if len(pts) == 0 {
		return 0
	}
	n := 1
	prev := pts[0]
	for i := 1; i < len(pts); i++ {
		dx := pts[i].X - prev.X
		dy := pts[i].Y - prev.Y
		if dx*dx+dy*dy > EpsilonBounds*EpsilonBounds {
			n++
			prev = pts[i]
		}
	}
	return n
}

// normalizeStrokes maps strokes into a unit box (legacy helper for geometry helpers / free-board).
func normalizeStrokes(strokes []Stroke) []Stroke {
	res := normalizeInk(strokes)
	if res.Empty {
		return cloneStrokes(strokes)
	}
	out := make([]Stroke, len(res.Strokes))
	for i, ns := range res.Strokes {
		out[i] = Stroke{Points: append([]Point(nil), ns.Points...)}
	}
	return out
}

func cloneStrokes(strokes []Stroke) []Stroke {
	out := make([]Stroke, len(strokes))
	for i, s := range strokes {
		out[i] = Stroke{Points: append([]Point(nil), s.Points...)}
	}
	return out
}
