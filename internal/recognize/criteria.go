package recognize

import "math"

// CriterionScores holds bounded [0,1] match components for one target comparison.
type CriterionScores struct {
	StrokeCount       float64
	StrokeOrder       float64
	StartEndDirection float64
	RelativePlacement float64
	Proportions       float64
	Shape             float64
}

// scoreCriteria compares already-normalized learner ink to a normalized template.
func scoreCriteria(learner, tmpl NormalizeResult) CriterionScores {
	got, want := len(learner.Strokes), len(tmpl.Strokes)
	cs := CriterionScores{
		StrokeCount: strokeCountScore(got, want),
		StrokeOrder: strokeOrderScore(learner.Strokes, tmpl.Strokes),
		StartEndDirection: directionScore(learner.Strokes, tmpl.Strokes),
		RelativePlacement: placementScore(learner.Strokes, tmpl.Strokes),
		Proportions:       proportionsScore(learner.Strokes, tmpl.Strokes),
		Shape:             shapeScore(learner.Strokes, tmpl.Strokes),
	}
	return cs
}

func weightedOverall(cs CriterionScores, w CriterionWeights) float64 {
	sum := w.StrokeCount*cs.StrokeCount +
		w.StrokeOrder*cs.StrokeOrder +
		w.StartEndDirection*cs.StartEndDirection +
		w.RelativePlacement*cs.RelativePlacement +
		w.Proportions*cs.Proportions +
		w.Shape*cs.Shape
	return clamp01(sum)
}

func strokeOrderScore(learner, tmpl []NormStroke) float64 {
	n := len(tmpl)
	if n == 0 {
		return 0
	}
	if len(learner) != n {
		// Count mismatch: order is unreliable — score by paired prefix identity only.
		pairs := len(learner)
		if pairs > n {
			pairs = n
		}
		if pairs == 0 {
			return 0
		}
		return clamp01(float64(pairs) / float64(n) * 0.35)
	}
	// Exhaustive best assignment by start-point distance (n ≤ 3 for MVP).
	bestCost := math.Inf(1)
	bestPerm := make([]int, n)
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	var search func(int, float64)
	search = func(k int, cost float64) {
		if k == n {
			if cost < bestCost {
				bestCost = cost
				copy(bestPerm, perm)
			}
			return
		}
		for i := k; i < n; i++ {
			perm[k], perm[i] = perm[i], perm[k]
			dx := learner[perm[k]].Start.X - tmpl[k].Start.X
			dy := learner[perm[k]].Start.Y - tmpl[k].Start.Y
			search(k+1, cost+math.Sqrt(dx*dx+dy*dy))
			perm[k], perm[i] = perm[i], perm[k]
		}
	}
	search(0, 0)
	correct := 0
	for k := 0; k < n; k++ {
		if bestPerm[k] == k {
			correct++
		}
	}
	return float64(correct) / float64(n)
}

func directionScore(learner, tmpl []NormStroke) float64 {
	n := len(tmpl)
	if n == 0 {
		return 0
	}
	pairs := n
	if len(learner) < pairs {
		pairs = len(learner)
	}
	sum := 0.0
	denom := 0
	for i := 0; i < pairs; i++ {
		a, b := learner[i], tmpl[i]
		if a.Kind == StrokeShort || b.Kind == StrokeShort {
			continue // skip dots / short strokes (D6)
		}
		startSim := tangentCosine(a.Points, true, b.Points, true)
		endSim := tangentCosine(a.Points, false, b.Points, false)
		sum += (startSim + endSim) / 2
		denom++
	}
	if denom == 0 {
		return 1 // only shorts / no pairs → N/A treated as full credit
	}
	// missing pairs beyond learner contribute 0
	return clamp01(sum / float64(n))
}

func tangentCosine(a []Point, atStart bool, b []Point, bAtStart bool) float64 {
	ua := unitTangent(a, atStart)
	ub := unitTangent(b, bAtStart)
	dot := ua.X*ub.X + ua.Y*ub.Y
	// Map cosine [-1,1] → [0,1]; soft floor around T_dir≈0.5 still continuous.
	return clamp01((dot + 1) / 2)
}

func unitTangent(pts []Point, atStart bool) Point {
	if len(pts) < 2 {
		return Point{X: 0, Y: 0}
	}
	var dx, dy float64
	if atStart {
		// First non-trivial segment.
		for i := 1; i < len(pts); i++ {
			dx = pts[i].X - pts[0].X
			dy = pts[i].Y - pts[0].Y
			if dx*dx+dy*dy > EpsilonBounds*EpsilonBounds {
				break
			}
		}
	} else {
		last := len(pts) - 1
		for i := last - 1; i >= 0; i-- {
			dx = pts[last].X - pts[i].X
			dy = pts[last].Y - pts[i].Y
			if dx*dx+dy*dy > EpsilonBounds*EpsilonBounds {
				break
			}
		}
	}
	mag := math.Sqrt(dx*dx + dy*dy)
	if mag < EpsilonBounds {
		return Point{}
	}
	return Point{X: dx / mag, Y: dy / mag}
}

func placementScore(learner, tmpl []NormStroke) float64 {
	n := len(tmpl)
	if n == 0 {
		return 0
	}
	pairs := n
	if len(learner) < pairs {
		pairs = len(learner)
	}
	if pairs == 0 {
		return 0
	}
	sum := 0.0
	for i := 0; i < pairs; i++ {
		dx := learner[i].Centroid.X - tmpl[i].Centroid.X
		dy := learner[i].Centroid.Y - tmpl[i].Centroid.Y
		sum += math.Sqrt(dx*dx + dy*dy)
	}
	// Missing template strokes count as PlaceFalloff distance.
	sum += float64(n-pairs) * PlaceFalloff
	mean := sum / float64(n)
	return clamp01(1 - mean/PlaceFalloff)
}

func proportionsScore(learner, tmpl []NormStroke) float64 {
	n := len(tmpl)
	if n == 0 {
		return 0
	}
	ltot := totalLength(learner)
	ttot := totalLength(tmpl)
	if ttot < EpsilonBounds {
		return 0
	}
	pairs := n
	if len(learner) < pairs {
		pairs = len(learner)
	}
	errSum := 0.0
	for i := 0; i < pairs; i++ {
		lr := 0.0
		if ltot > EpsilonBounds {
			lr = learner[i].Length / ltot
		}
		tr := tmpl[i].Length / ttot
		diff := math.Abs(lr - tr)
		errSum += diff
	}
	errSum += float64(n - pairs) // missing → full ratio error
	meanErr := errSum / float64(n)

	// Overall ink aspect (bbox of all points).
	aspectErr := math.Abs(inkAspect(learner) - inkAspect(tmpl))
	combined := 0.7*meanErr + 0.3*aspectErr
	return clamp01(1 - combined/PropFalloff)
}

func totalLength(strokes []NormStroke) float64 {
	s := 0.0
	for _, st := range strokes {
		s += st.Length
	}
	return s
}

func inkAspect(strokes []NormStroke) float64 {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	any := false
	for _, st := range strokes {
		for _, p := range st.Points {
			any = true
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
	if !any {
		return 1
	}
	w := maxX - minX
	h := maxY - minY
	if h < EpsilonBounds {
		return 1
	}
	return w / h
}

func shapeScore(learner, tmpl []NormStroke) float64 {
	n := len(tmpl)
	if n == 0 {
		return 0
	}
	pairs := n
	if len(learner) < pairs {
		pairs = len(learner)
	}
	if pairs == 0 {
		return 0
	}
	sum := 0.0
	for i := 0; i < pairs; i++ {
		a, b := learner[i], tmpl[i]
		if a.Kind == StrokeShort || b.Kind == StrokeShort {
			// Centroid proximity for shorts (light shape contribution).
			dx := a.Centroid.X - b.Centroid.X
			dy := a.Centroid.Y - b.Centroid.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			sum += clamp01(1 - dist/PlaceFalloff)
			continue
		}
		sum += strokeSimilarity(Stroke{Points: a.Points}, Stroke{Points: b.Points})
	}
	extra := math.Abs(float64(len(learner) - len(tmpl)))
	penalty := clamp01(extra / float64(n))
	avg := sum / float64(n)
	return clamp01(avg * (1 - 0.5*penalty))
}

// strokeHardFail reports exact count mismatch for small MVP glyphs (want ≤ HardFailMaxStrokeCount).
func strokeHardFail(got, want int) bool {
	return want > 0 && want <= HardFailMaxStrokeCount && got != want
}
