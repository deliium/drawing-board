package recognize

import "errors"

// Recognizer ranks free-board strokes with heuristic match scores (not calibrated confidence).
type Recognizer interface {
	Recognize(strokes []Stroke, width, height int, topN int) ([]Candidate, error)
	Close() error
}

// Assessor compares learner strokes to a known target from the MVP set.
type Assessor interface {
	Assess(target string, strokes []Stroke, width, height int) (Assessment, error)
}

// ScoreKindMatch labels Candidate.Score / Assessment.Score as a heuristic match score.
const ScoreKindMatch = "match"

// SetIDHiragana5 is the fixed five-character MVP curriculum id (あ行 vowels).
const SetIDHiragana5 = "hiragana5"

// PassThreshold (T_pass) is the engineering pass bar for gold fixtures / target assessment.
const PassThreshold = 0.70

// ErrUnsupportedTarget is returned when Assess receives a glyph outside the MVP set.
var ErrUnsupportedTarget = errors.New("unsupported_target")

// Types for stroke recognition.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type Stroke struct {
	Points []Point `json:"points"`
}

// Candidate is a ranked suggestion. Score is a match score (scoreKind=match), never calibrated confidence.
type Candidate struct {
	Text      string  `json:"text"`
	Score     float64 `json:"score"`
	ScoreKind string  `json:"scoreKind,omitempty"`
}

// Diagnostics holds engineer-facing criterion breakdown (not learner copy).
type Diagnostics struct {
	StrokeCount       float64 `json:"strokeCount"`
	StrokeOrder       float64 `json:"strokeOrder"`
	StartEndDirection float64 `json:"startEndDirection"`
	RelativePlacement float64 `json:"relativePlacement"`
	Proportions       float64 `json:"proportions"`
	Shape             float64 `json:"shape"`
	Overall           float64 `json:"overall"`
	HardFail          bool    `json:"hardFail"`
	HardFailCode      string  `json:"hardFailCode,omitempty"`
	ShortStrokeCount  int     `json:"shortStrokeCount"`
	NormalStrokeCount int     `json:"normalStrokeCount"`
	DroppedEmpty      int     `json:"droppedEmpty"`
}

// Assessment is the result of comparing strokes to a known target character.
type Assessment struct {
	Target      string         `json:"target"`
	Pass        bool           `json:"pass"`
	Score       float64        `json:"score"`
	ScoreKind   string         `json:"scoreKind"`
	Reasons     []string       `json:"reasons"`
	Feedback    []FeedbackItem `json:"feedback,omitempty"`
	Diagnostics *Diagnostics   `json:"diagnostics,omitempty"`
	Candidates  []Candidate    `json:"candidates,omitempty"`
}
