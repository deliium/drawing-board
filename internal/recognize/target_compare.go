package recognize

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/deliium/drawing-board/internal/curriculum"
)

// TargetCompareRecognizer scores learner strokes against hiragana5 templates.
// Free-board Recognize delegates to SimpleRecognizer (heuristic match scores).
type TargetCompareRecognizer struct {
	version   string
	setID     string
	templates map[string][]Stroke
	order     []string
	simple    *SimpleRecognizer
}

// NewTargetCompareRecognizer loads assessment strokes from the published curriculum pack.
func NewTargetCompareRecognizer() (*TargetCompareRecognizer, error) {
	pack, err := curriculum.LoadPublishedV1()
	if err != nil {
		log.Printf("ERROR [recognize.templates] load pack failed: %v", err)
		return nil, fmt.Errorf("load hiragana5 pack: %w", err)
	}
	if len(pack.Chars) != 5 {
		err := fmt.Errorf("hiragana5 pack: want 5 characters, got %d", len(pack.Chars))
		log.Printf("ERROR [recognize.templates] %v", err)
		return nil, err
	}
	r := &TargetCompareRecognizer{
		version:   pack.Manifest.ContentVersion,
		setID:     pack.Manifest.SetID,
		templates: make(map[string][]Stroke, 5),
		order:     make([]string, 0, 5),
		simple:    NewSimpleRecognizer(),
	}
	if r.setID == "" {
		r.setID = SetIDHiragana5
	}
	chars := append([]curriculum.Character(nil), pack.Chars...)
	sort.Slice(chars, func(i, j int) bool { return chars[i].SortKey < chars[j].SortKey })
	for _, ch := range chars {
		raw := pack.Strokes[ch.Glyph]
		if ch.Glyph == "" || len(raw) == 0 {
			err := fmt.Errorf("invalid template for glyph %q", ch.Glyph)
			log.Printf("ERROR [recognize.templates] %v", err)
			return nil, err
		}
		strokes := make([]Stroke, len(raw))
		for i, pts := range raw {
			points := make([]Point, len(pts))
			for j, pt := range pts {
				points[j] = Point{X: pt.X, Y: pt.Y}
			}
			strokes[i] = Stroke{Points: points}
		}
		r.templates[ch.Glyph] = strokes
		r.order = append(r.order, ch.Glyph)
	}
	log.Printf("DEBUG [recognize.templates] loaded count=%d version=%s set=%s", len(r.order), r.version, r.setID)
	return r, nil
}

func (r *TargetCompareRecognizer) Close() error {
	if r.simple != nil {
		return r.simple.Close()
	}
	return nil
}

// SetID returns the curriculum set identifier.
func (r *TargetCompareRecognizer) SetID() string { return r.setID }

// Version returns the template pack version.
func (r *TargetCompareRecognizer) Version() string { return r.version }

// Recognize performs free-board heuristic ranking via SimpleRecognizer with scoreKind=match.
func (r *TargetCompareRecognizer) Recognize(strokes []Stroke, width, height int, topN int) ([]Candidate, error) {
	pointCount := countPoints(strokes)
	logf("DEBUG", "[recognize.Recognize] mode=heuristic strokes=%d points=%d width=%d height=%d topN=%d",
		len(strokes), pointCount, width, height, topN)
	cands, err := r.simple.Recognize(strokes, width, height, topN)
	if err != nil {
		return nil, err
	}
	for i := range cands {
		cands[i].ScoreKind = ScoreKindMatch
	}
	logf("DEBUG", "[recognize.Recognize] mode=heuristic candidates=%d", len(cands))
	return cands, nil
}

// Assess compares strokes to a known MVP target using multi-criterion match scoring.
func (r *TargetCompareRecognizer) Assess(target string, strokes []Stroke, width, height int) (Assessment, error) {
	if err := validateCanvasOnly(width, height); err != nil {
		return Assessment{}, err
	}
	target = strings.TrimSpace(target)
	tmplRaw, ok := r.templates[target]
	if !ok {
		logf("WARN", "[recognize.Assess] unsupported_target=%q", target)
		return Assessment{}, fmt.Errorf("%w: %s", ErrUnsupportedTarget, target)
	}

	pointCount := countPoints(strokes)
	logf("DEBUG", "[recognize.Assess] target=%s strokes=%d points=%d width=%d height=%d",
		target, len(strokes), pointCount, width, height)

	learnerNorm := normalizeInk(strokes)
	tmplNorm := normalizeInk(tmplRaw)
	want := len(tmplNorm.Strokes)

	if learnerNorm.Empty {
		fb := selectFeedback(target, false, 0, want, true, false, CriterionScores{}, nil, nil)
		out := Assessment{
			Target:    target,
			Pass:      false,
			Score:     0,
			ScoreKind: ScoreKindMatch,
			Reasons:   []string{CodeEmptyStrokes},
			Feedback:  fb,
			Diagnostics: &Diagnostics{
				DroppedEmpty: learnerNorm.DroppedEmpty,
				HardFail:     false,
			},
		}
		logf("DEBUG", "[recognize.Assess] target=%s pass=false score=0 reasons=empty_strokes feedback=%d", target, len(fb))
		return out, nil
	}

	cs := scoreCriteria(learnerNorm, tmplNorm)
	overall := weightedOverall(cs, DefaultWeights)
	got := len(learnerNorm.Strokes)
	hardFail := strokeHardFail(got, want)
	hardCode := ""
	if hardFail {
		hardCode = CodeStrokeCountMismatch
	}
	ranked := r.rankAll(strokes)
	topMatch := len(ranked) > 0 && ranked[0].Text == target
	// Pass requires T_pass, no stroke-count hard-fail, and target top-ranked in the MVP set
	// (similar wrong-glyph templates can otherwise clear T_pass alone).
	pass := overall >= PassThreshold && !hardFail && topMatch

	diag := &Diagnostics{
		StrokeCount:       cs.StrokeCount,
		StrokeOrder:       cs.StrokeOrder,
		StartEndDirection: cs.StartEndDirection,
		RelativePlacement: cs.RelativePlacement,
		Proportions:       cs.Proportions,
		Shape:             cs.Shape,
		Overall:           overall,
		HardFail:          hardFail,
		HardFailCode:      hardCode,
		ShortStrokeCount:  learnerNorm.ShortStrokeCount,
		NormalStrokeCount: learnerNorm.NormalStrokeCount,
		DroppedEmpty:      learnerNorm.DroppedEmpty,
	}

	feedback := selectFeedback(target, pass, got, want, false, hardFail, cs, learnerNorm.Strokes, tmplNorm.Strokes)
	// Prefer shape coaching when another MVP glyph outranks the target (D7).
	if !pass && !hardFail && !topMatch {
		feedback = ensureFeedbackCode(feedback, CodeShape, target, want, got)
	}
	reasons := buildReasons(cs, hardFail, pass, got, want)
	if !pass && overall >= PassThreshold && !hardFail && !topMatch {
		reasons = append(reasons, "outranked_by_other")
	}

	out := Assessment{
		Target:      target,
		Pass:        pass,
		Score:       overall,
		ScoreKind:   ScoreKindMatch,
		Reasons:     reasons,
		Feedback:    feedback,
		Diagnostics: diag,
		Candidates:  ranked,
	}
	if hardFail && len(feedback) == 0 {
		logf("WARN", "[recognize.Assess] hard_fail_without_feedback target=%s", target)
	}
	codes := make([]string, len(feedback))
	for i, f := range feedback {
		codes[i] = f.Code
	}
	logf("DEBUG", "[recognize.Assess] target=%s pass=%t score=%.3f criteria={count=%.3f order=%.3f dir=%.3f place=%.3f prop=%.3f shape=%.3f} hardFail=%t short=%d normal=%d feedback=%v",
		target, pass, overall, cs.StrokeCount, cs.StrokeOrder, cs.StartEndDirection, cs.RelativePlacement, cs.Proportions, cs.Shape,
		hardFail, learnerNorm.ShortStrokeCount, learnerNorm.NormalStrokeCount, codes)
	return out, nil
}

func buildReasons(cs CriterionScores, hardFail, pass bool, got, want int) []string {
	reasons := make([]string, 0, 8)
	if got != want {
		reasons = append(reasons, CodeStrokeCountMismatch)
	}
	if hardFail {
		reasons = append(reasons, "hard_fail_stroke_count")
	}
	if cs.StrokeOrder < SoftCriterionThreshold {
		reasons = append(reasons, CodeStrokeOrder)
	}
	if cs.StartEndDirection < SoftCriterionThreshold {
		reasons = append(reasons, CodeStartDirection)
	}
	if cs.RelativePlacement < SoftCriterionThreshold {
		reasons = append(reasons, CodeRelativePlacement)
	}
	if cs.Proportions < SoftCriterionThreshold {
		reasons = append(reasons, CodeProportions)
	}
	if cs.Shape < SoftCriterionThreshold {
		reasons = append(reasons, CodeShape)
	}
	if pass {
		reasons = append(reasons, "pass")
	}
	return reasons
}

func (r *TargetCompareRecognizer) rankAll(strokes []Stroke) []Candidate {
	learner := normalizeInk(strokes)
	cands := make([]Candidate, 0, len(r.order))
	for _, glyph := range r.order {
		tmpl := normalizeInk(r.templates[glyph])
		cs := scoreCriteria(learner, tmpl)
		score := weightedOverall(cs, DefaultWeights)
		cands = append(cands, Candidate{Text: glyph, Score: score, ScoreKind: ScoreKindMatch})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].Score == cands[j].Score {
			return cands[i].Text < cands[j].Text
		}
		return cands[i].Score > cands[j].Score
	})
	return cands
}

func validateCanvasOnly(width, height int) error {
	_, err := validateRecognizeParams(width, height, 0)
	return err
}

func countPoints(strokes []Stroke) int {
	n := 0
	for _, s := range strokes {
		n += len(s.Points)
	}
	return n
}

func strokeCountScore(got, want int) float64 {
	if want <= 0 {
		return 0
	}
	if got == want {
		return 1
	}
	diff := math.Abs(float64(got - want))
	return clamp01(1 - diff/float64(want))
}

func geometryScore(learner, template []Stroke) float64 {
	n := len(template)
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
		sum += strokeSimilarity(learner[i], template[i])
	}
	extra := math.Abs(float64(len(learner) - len(template)))
	penalty := clamp01(extra / float64(n))
	avg := sum / float64(n)
	return clamp01(avg * (1 - 0.5*penalty))
}

func strokeSimilarity(a, b Stroke) float64 {
	ra := resampleStroke(a, ResampleCount)
	rb := resampleStroke(b, ResampleCount)
	if len(ra) == 0 || len(rb) == 0 {
		if len(ra) == 0 && len(rb) == 0 {
			return 1
		}
		return 0
	}
	dist := 0.0
	for i := 0; i < ResampleCount; i++ {
		dx := ra[i].X - rb[i].X
		dy := ra[i].Y - rb[i].Y
		dist += math.Sqrt(dx*dx + dy*dy)
	}
	avg := dist / float64(ResampleCount)
	return clamp01(1 - avg/0.5)
}

func resampleStroke(s Stroke, n int) []Point {
	if n <= 0 {
		return nil
	}
	if len(s.Points) == 0 {
		return make([]Point, n)
	}
	if len(s.Points) == 1 {
		out := make([]Point, n)
		for i := range out {
			out[i] = s.Points[0]
		}
		return out
	}
	dists := make([]float64, len(s.Points))
	total := 0.0
	for i := 1; i < len(s.Points); i++ {
		dx := s.Points[i].X - s.Points[i-1].X
		dy := s.Points[i].Y - s.Points[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
		dists[i] = total
	}
	if total < EpsilonBounds {
		out := make([]Point, n)
		for i := range out {
			out[i] = s.Points[0]
		}
		return out
	}
	out := make([]Point, n)
	for i := 0; i < n; i++ {
		target := total * float64(i) / float64(n-1)
		j := 1
		for j < len(dists) && dists[j] < target {
			j++
		}
		if j >= len(s.Points) {
			out[i] = s.Points[len(s.Points)-1]
			continue
		}
		prev := dists[j-1]
		seg := dists[j] - prev
		t := 0.0
		if seg > EpsilonBounds {
			t = (target - prev) / seg
		}
		p0 := s.Points[j-1]
		p1 := s.Points[j]
		out[i] = Point{X: p0.X + t*(p1.X-p0.X), Y: p0.Y + t*(p1.Y-p0.Y)}
	}
	return out
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func logf(level, format string, args ...interface{}) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "error":
		if level != "ERROR" {
			return
		}
	case "warn":
		if level != "ERROR" && level != "WARN" {
			return
		}
	case "info":
		if level == "DEBUG" {
			return
		}
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s %s", level, msg)
}
