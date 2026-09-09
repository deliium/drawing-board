package recognize

// Engineering tolerances and criterion weights for hiragana5 target assessment.
// These are match-score thresholds — not calibrated accuracy percentages.

const (
	// EpsilonBounds clamps degenerate bbox axes after whole-ink bounds.
	EpsilonBounds = 1e-9
	// ShortStrokeLength (L_short) in unit space: path length ≤ this (or ≤1 distinct point) → short/dot.
	ShortStrokeLength = 0.04
	// ResampleCount (N_resample) for shape and direction sampling on normal strokes.
	ResampleCount = 16
	// SoftCriterionThreshold: criterion scores below this may emit a learner correction.
	SoftCriterionThreshold = 0.75
	// PlaceFalloff (T_place): mean centroid distance at this value → placement score ~0.
	PlaceFalloff = 0.15
	// PropFalloff (T_prop): relative length-ratio error at this value → proportions score ~0.
	PropFalloff = 0.35
	// HardFailMaxStrokeCount: exact stroke-count mismatch hard-fails when want ≤ this (MVP glyphs are 2–3).
	HardFailMaxStrokeCount = 3
)

// CriterionWeights are the published overall-score weights (must sum to 1).
type CriterionWeights struct {
	StrokeCount       float64
	StrokeOrder       float64
	StartEndDirection float64
	RelativePlacement float64
	Proportions       float64
	Shape             float64
}

// DefaultWeights is the hiragana5 MVP weight table.
var DefaultWeights = CriterionWeights{
	StrokeCount:       0.15,
	StrokeOrder:       0.15,
	StartEndDirection: 0.15,
	RelativePlacement: 0.15,
	Proportions:       0.10,
	Shape:             0.30,
}
