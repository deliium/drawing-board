package recognize

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"embed"
)

//go:embed testdata/hiragana5/templates.json
var templatesFS embed.FS

const (
	weightStrokeCount = 0.35
	weightGeometry    = 0.65
	resamplePoints    = 16
)

type templateFile struct {
	Version    string             `json:"version"`
	SetID      string             `json:"setId"`
	Characters []templateCharacter `json:"characters"`
}

type templateCharacter struct {
	Glyph   string     `json:"glyph"`
	Strokes [][]Point  `json:"strokes"`
}

// TargetCompareRecognizer scores learner strokes against hiragana5 templates.
// Free-board Recognize delegates to SimpleRecognizer (heuristic match scores).
type TargetCompareRecognizer struct {
	version   string
	setID     string
	templates map[string][]Stroke
	order     []string
	simple    *SimpleRecognizer
}

// NewTargetCompareRecognizer loads the embedded hiragana5 templates.
func NewTargetCompareRecognizer() (*TargetCompareRecognizer, error) {
	raw, err := templatesFS.ReadFile("testdata/hiragana5/templates.json")
	if err != nil {
		log.Printf("ERROR [recognize.templates] load failed: %v", err)
		return nil, fmt.Errorf("load hiragana5 templates: %w", err)
	}
	var tf templateFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		log.Printf("ERROR [recognize.templates] parse failed: %v", err)
		return nil, fmt.Errorf("parse hiragana5 templates: %w", err)
	}
	if len(tf.Characters) != 5 {
		err := fmt.Errorf("hiragana5 templates: want 5 characters, got %d", len(tf.Characters))
		log.Printf("ERROR [recognize.templates] %v", err)
		return nil, err
	}
	r := &TargetCompareRecognizer{
		version:   tf.Version,
		setID:     tf.SetID,
		templates: make(map[string][]Stroke, 5),
		order:     make([]string, 0, 5),
		simple:    NewSimpleRecognizer(),
	}
	if r.setID == "" {
		r.setID = SetIDHiragana5
	}
	for _, ch := range tf.Characters {
		if ch.Glyph == "" || len(ch.Strokes) == 0 {
			err := fmt.Errorf("invalid template for glyph %q", ch.Glyph)
			log.Printf("ERROR [recognize.templates] %v", err)
			return nil, err
		}
		strokes := make([]Stroke, len(ch.Strokes))
		for i, pts := range ch.Strokes {
			strokes[i] = Stroke{Points: append([]Point(nil), pts...)}
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

// Assess compares strokes to a known MVP target. Empty strokes → non-pass, empty candidates.
func (r *TargetCompareRecognizer) Assess(target string, strokes []Stroke, width, height int) (Assessment, error) {
	if err := validateCanvasOnly(width, height); err != nil {
		return Assessment{}, err
	}
	target = strings.TrimSpace(target)
	tmpl, ok := r.templates[target]
	if !ok {
		logf("WARN", "[recognize.Assess] unsupported_target=%q", target)
		return Assessment{}, fmt.Errorf("%w: %s", ErrUnsupportedTarget, target)
	}

	pointCount := countPoints(strokes)
	logf("DEBUG", "[recognize.Assess] target=%s strokes=%d points=%d width=%d height=%d",
		target, len(strokes), pointCount, width, height)

	if len(strokes) == 0 {
		out := Assessment{
			Target:    target,
			Pass:      false,
			Score:     0,
			ScoreKind: ScoreKindMatch,
			Reasons:   []string{"empty_strokes"},
		}
		logf("DEBUG", "[recognize.Assess] target=%s pass=false score=0 reasons=empty_strokes", target)
		return out, nil
	}

	ranked := r.rankAll(strokes)
	var targetScore float64
	var reasons []string
	for _, c := range ranked {
		if c.Text == target {
			targetScore = c.Score
			break
		}
	}
	countScore := strokeCountScore(len(strokes), len(tmpl))
	geoScore := geometryScore(normalizeStrokes(strokes), normalizeStrokes(tmpl))
	reasons = append(reasons,
		fmt.Sprintf("stroke_count=%.3f", countScore),
		fmt.Sprintf("geometry=%.3f", geoScore),
		fmt.Sprintf("combined=%.3f", targetScore),
	)
	if len(strokes) != len(tmpl) {
		reasons = append(reasons, "stroke_count_mismatch")
	}

	pass := targetScore >= PassThreshold && ranked[0].Text == target
	if !pass && targetScore >= PassThreshold && ranked[0].Text != target {
		reasons = append(reasons, "outranked_by_other")
	}
	if pass {
		reasons = append(reasons, "top_match")
	}

	out := Assessment{
		Target:     target,
		Pass:       pass,
		Score:      targetScore,
		ScoreKind:  ScoreKindMatch,
		Reasons:    reasons,
		Candidates: ranked,
	}
	logf("DEBUG", "[recognize.Assess] target=%s pass=%t score=%.3f reasons=%v", target, pass, targetScore, reasons)
	return out, nil
}

func (r *TargetCompareRecognizer) rankAll(strokes []Stroke) []Candidate {
	learner := normalizeStrokes(strokes)
	cands := make([]Candidate, 0, len(r.order))
	for _, glyph := range r.order {
		tmpl := normalizeStrokes(r.templates[glyph])
		count := strokeCountScore(len(strokes), len(r.templates[glyph]))
		geo := geometryScore(learner, tmpl)
		score := weightStrokeCount*count + weightGeometry*geo
		cands = append(cands, Candidate{Text: glyph, Score: clamp01(score), ScoreKind: ScoreKindMatch})
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

// normalizeStrokes maps strokes into a unit box with aspect ratio preserved (letterbox).
func normalizeStrokes(strokes []Stroke) []Stroke {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	any := false
	for _, s := range strokes {
		for _, p := range s.Points {
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
		return cloneStrokes(strokes)
	}
	w := maxX - minX
	h := maxY - minY
	if w < 1e-9 {
		w = 1
	}
	if h < 1e-9 {
		h = 1
	}
	scale := 1 / math.Max(w, h)
	out := make([]Stroke, len(strokes))
	for i, s := range strokes {
		pts := make([]Point, len(s.Points))
		for j, p := range s.Points {
			pts[j] = Point{
				X: (p.X - minX) * scale,
				Y: (p.Y - minY) * scale,
			}
		}
		out[i] = Stroke{Points: pts}
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
	// Penalize extra/missing strokes beyond paired ones.
	extra := math.Abs(float64(len(learner) - len(template)))
	penalty := clamp01(extra / float64(n))
	avg := sum / float64(n) // missing pairs contribute 0
	return clamp01(avg * (1 - 0.5*penalty))
}

func strokeSimilarity(a, b Stroke) float64 {
	ra := resampleStroke(a, resamplePoints)
	rb := resampleStroke(b, resamplePoints)
	if len(ra) == 0 || len(rb) == 0 {
		if len(ra) == 0 && len(rb) == 0 {
			return 1
		}
		return 0
	}
	dist := 0.0
	for i := 0; i < resamplePoints; i++ {
		dx := ra[i].X - rb[i].X
		dy := ra[i].Y - rb[i].Y
		dist += math.Sqrt(dx*dx + dy*dy)
	}
	avg := dist / float64(resamplePoints)
	// avg distance 0 → 1; ~0.5 unit → ~0
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
	// Cumulative length.
	dists := make([]float64, len(s.Points))
	total := 0.0
	for i := 1; i < len(s.Points); i++ {
		dx := s.Points[i].X - s.Points[i-1].X
		dy := s.Points[i].Y - s.Points[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
		dists[i] = total
	}
	if total < 1e-9 {
		out := make([]Point, n)
		for i := range out {
			out[i] = s.Points[0]
		}
		return out
	}
	out := make([]Point, n)
	for i := 0; i < n; i++ {
		target := total * float64(i) / float64(n-1)
		// Find segment.
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
		if seg > 1e-9 {
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
