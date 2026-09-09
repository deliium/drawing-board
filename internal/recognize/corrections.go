package recognize

import (
	"fmt"
	"log"
	"strings"
)

// FeedbackItem is one ranked learner-facing correction (≤2 emitted per assessment).
type FeedbackItem struct {
	Rank    int    `json:"rank"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Correction codes (stable API / persistence).
const (
	CodeEmptyStrokes        = "empty_strokes"
	CodeStrokeCountMismatch = "stroke_count_mismatch"
	CodeStrokeOrder         = "stroke_order"
	CodeStartDirection      = "start_direction"
	CodeEndDirection        = "end_direction"
	CodeRelativePlacement   = "relative_placement"
	CodeProportions         = "proportions"
	CodeShape               = "shape"
)

// correctionPriority is highest-first selection order for ≤2 feedback items.
var correctionPriority = []string{
	CodeEmptyStrokes,
	CodeStrokeCountMismatch,
	CodeStrokeOrder,
	CodeStartDirection,
	CodeEndDirection,
	CodeRelativePlacement,
	CodeProportions,
	CodeShape,
}

type correctionCandidate struct {
	code    string
	strokeN int // 1-based stroke index when relevant; 0 = N/A
}

func catalogMessage(code, glyph string, want, got, strokeN int) string {
	switch code {
	case CodeEmptyStrokes:
		return "No strokes were submitted. Draw the character, then try again."
	case CodeStrokeCountMismatch:
		return fmt.Sprintf("Use %d strokes for 「%s」 (you used %d). Assessed attempts are immutable — start a new attempt and focus on stroke count.", want, glyph, got)
	case CodeStrokeOrder:
		return fmt.Sprintf("Check stroke order — start with the stroke that begins at the top/left for 「%s」. Start a new attempt after this one.", glyph)
	case CodeStartDirection:
		if strokeN > 0 {
			return fmt.Sprintf("Start stroke %d in the same direction as the model (see the tip of the first movement). Retry on a new attempt.", strokeN)
		}
		return "Start the stroke in the same direction as the model. Retry on a new attempt."
	case CodeEndDirection:
		if strokeN > 0 {
			return fmt.Sprintf("Finish stroke %d in the expected direction. Retry on a new attempt.", strokeN)
		}
		return "Finish the stroke in the expected direction. Retry on a new attempt."
	case CodeRelativePlacement:
		return "Place the strokes closer to their usual positions relative to each other. Start a new attempt focusing on placement."
	case CodeProportions:
		return fmt.Sprintf("Adjust the length or size of the strokes so parts of 「%s」 match usual proportions. Retry on a new attempt.", glyph)
	case CodeShape:
		return fmt.Sprintf("The overall shape of 「%s」 still differs from the model — slow down and retrace on a new attempt.", glyph)
	default:
		log.Printf("WARN [recognize.corrections] unknown_code=%q", code)
		return "Review the model character and try again on a new attempt."
	}
}

// selectFeedback picks at most two learner corrections from criterion weakness / hard-fail.
func selectFeedback(glyph string, pass bool, got, want int, empty bool, hardFail bool, cs CriterionScores, learner, tmpl []NormStroke) []FeedbackItem {
	if empty {
		return []FeedbackItem{{
			Rank:    1,
			Code:    CodeEmptyStrokes,
			Message: catalogMessage(CodeEmptyStrokes, glyph, want, got, 0),
		}}
	}

	cands := make([]correctionCandidate, 0, 8)
	if hardFail || got != want || cs.StrokeCount < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeStrokeCountMismatch})
	}
	if got == want && cs.StrokeOrder < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeStrokeOrder})
	}
	if startN, endN, startWeak, endWeak := directionWeakness(learner, tmpl); startWeak || endWeak {
		if startWeak {
			cands = append(cands, correctionCandidate{code: CodeStartDirection, strokeN: startN})
		}
		if endWeak {
			cands = append(cands, correctionCandidate{code: CodeEndDirection, strokeN: endN})
		}
	} else if cs.StartEndDirection < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeStartDirection, strokeN: 1})
	}
	if cs.RelativePlacement < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeRelativePlacement})
	}
	if cs.Proportions < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeProportions})
	}
	if cs.Shape < SoftCriterionThreshold {
		cands = append(cands, correctionCandidate{code: CodeShape})
	}

	if pass && len(cands) == 0 {
		return nil
	}
	// Non-pass with strokes should usually get ≥1 feedback (D7).
	if !pass && len(cands) == 0 {
		cands = append(cands, correctionCandidate{code: CodeShape})
	}

	chosen := pickByPriority(cands, 2)
	out := make([]FeedbackItem, 0, len(chosen))
	for i, c := range chosen {
		out = append(out, FeedbackItem{
			Rank:    i + 1,
			Code:    c.code,
			Message: catalogMessage(c.code, glyph, want, got, c.strokeN),
		})
	}
	codes := make([]string, len(out))
	for i, f := range out {
		codes[i] = f.Code
	}
	logf("DEBUG", "[recognize.corrections] selected=%s", strings.Join(codes, ","))
	return out
}

func pickByPriority(cands []correctionCandidate, max int) []correctionCandidate {
	seen := map[string]correctionCandidate{}
	for _, c := range cands {
		if _, ok := seen[c.code]; !ok {
			seen[c.code] = c
		}
	}
	out := make([]correctionCandidate, 0, max)
	for _, code := range correctionPriority {
		if c, ok := seen[code]; ok {
			out = append(out, c)
			if len(out) == max {
				break
			}
		}
	}
	return out
}

// ensureFeedbackCode inserts code into the ≤2 feedback list (displacing the lowest-priority item if full).
func ensureFeedbackCode(fb []FeedbackItem, code, glyph string, want, got int) []FeedbackItem {
	for _, item := range fb {
		if item.Code == code {
			return fb
		}
	}
	item := FeedbackItem{Code: code, Message: catalogMessage(code, glyph, want, got, 0)}
	if len(fb) < 2 {
		item.Rank = len(fb) + 1
		return append(fb, item)
	}
	// Replace lowest-priority among current two if the ensured code ranks higher.
	prio := map[string]int{}
	for i, c := range correctionPriority {
		prio[c] = i
	}
	ensuredPrio, ok := prio[code]
	if !ok {
		ensuredPrio = len(correctionPriority)
	}
	replace := 1
	if prio[fb[0].Code] > prio[fb[1].Code] {
		replace = 0
	}
	if ensuredPrio < prio[fb[replace].Code] || ensuredPrio <= prio[fb[1].Code] {
		fb[replace] = item
	} else {
		// Still force pedagogical shape into slot 2 for outranked cases.
		fb[1] = item
	}
	for i := range fb {
		fb[i].Rank = i + 1
	}
	return fb
}

// directionWeakness finds the worst start/end tangent mismatch among paired normal strokes.
func directionWeakness(learner, tmpl []NormStroke) (startN, endN int, startWeak, endWeak bool) {
	pairs := len(tmpl)
	if len(learner) < pairs {
		pairs = len(learner)
	}
	worstStart, worstEnd := 1.0, 1.0
	for i := 0; i < pairs; i++ {
		a, b := learner[i], tmpl[i]
		if a.Kind == StrokeShort || b.Kind == StrokeShort {
			continue
		}
		s := tangentCosine(a.Points, true, b.Points, true)
		e := tangentCosine(a.Points, false, b.Points, false)
		if s < worstStart {
			worstStart = s
			startN = i + 1
		}
		if e < worstEnd {
			worstEnd = e
			endN = i + 1
		}
	}
	// SoftCriterionThreshold is on [0,1] match; tangentCosine already maps cosine.
	startWeak = startN > 0 && worstStart < SoftCriterionThreshold
	endWeak = endN > 0 && worstEnd < SoftCriterionThreshold
	return startN, endN, startWeak, endWeak
}
