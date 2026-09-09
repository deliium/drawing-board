package learn

import "time"

// Mastery states — engineering heuristic from assessed attempts only (not SRS / proficiency science).
const (
	MasteryStateNotStarted = "not_started"
	MasteryStateLearning   = "learning"
	MasteryStatePassedOnce = "passed_once"
	MasteryStateSteady     = "steady"
)

// Mastery reason codes for i18n (stable snake_case).
const (
	MasteryReasonNoAssessedAttempts  = "no_assessed_attempts"
	MasteryReasonNoPassesYet         = "no_passes_yet"
	MasteryReasonSinglePass          = "single_pass"
	MasteryReasonRecentMiss          = "recent_miss"
	MasteryReasonTwoConsecutivePasses = "two_consecutive_passes"
)

// Next-character suggestion reason codes (humble practice order — no due dates).
const (
	NextReasonFirstNotStarted = "first_not_started"
	NextReasonContinueLearning = "continue_learning"
	NextReasonEncourageSteady = "encourage_steady"
	NextReasonAllSteady       = "all_steady"
)

// MasteryOutcomeWindow caps assessed attempts used for compute-on-read mastery facts.
const MasteryOutcomeWindow = 20

// AssessedOutcome is one assessed attempt timeline point (pass/fail only; no scores).
type AssessedOutcome struct {
	AttemptID  int64
	Pass       bool
	AssessedAt time.Time
}

// Mastery is a derived practice summary — not a validated proficiency or ML score.
type Mastery struct {
	State                   string
	ReasonCode              string
	AssessedCount           int
	PassCount               int
	FailCount               int
	ConsecutivePassesEnding int
	LastPass                *bool
	LastAssessedAt          *time.Time
}

// DeriveMastery computes explainable mastery from assessed outcomes only.
//
// Callers must exclude abandoned and submitted-only attempts. Outcomes should be
// ordered by AssessedAt ASC, AttemptID ASC for ties. This is an engineering
// heuristic for personal practice summaries — not SRS, belts, or calibrated confidence.
func DeriveMastery(outcomes []AssessedOutcome) Mastery {
	m := Mastery{
		State:      MasteryStateNotStarted,
		ReasonCode: MasteryReasonNoAssessedAttempts,
	}
	if len(outcomes) == 0 {
		return m
	}

	m.AssessedCount = len(outcomes)
	passCount := 0
	for _, o := range outcomes {
		if o.Pass {
			passCount++
		}
	}
	m.PassCount = passCount
	m.FailCount = m.AssessedCount - passCount

	last := outcomes[len(outcomes)-1]
	lastPass := last.Pass
	m.LastPass = &lastPass
	t := last.AssessedAt
	m.LastAssessedAt = &t

	consec := 0
	for i := len(outcomes) - 1; i >= 0; i-- {
		if !outcomes[i].Pass {
			break
		}
		consec++
	}
	m.ConsecutivePassesEnding = consec

	switch {
	case m.PassCount == 0:
		m.State = MasteryStateLearning
		m.ReasonCode = MasteryReasonNoPassesYet
	case m.ConsecutivePassesEnding >= 2:
		m.State = MasteryStateSteady
		m.ReasonCode = MasteryReasonTwoConsecutivePasses
	default:
		m.State = MasteryStatePassedOnce
		if lastPass {
			m.ReasonCode = MasteryReasonSinglePass
		} else {
			m.ReasonCode = MasteryReasonRecentMiss
		}
	}
	return m
}

// SuggestNextCharacter picks the first lesson character needing practice by mastery state priority.
// Priority: not_started → learning → passed_once → all_steady (nil suggestion).
// lessonOrder is lesson position order (already sorted). masteryByCharacter maps characterID → mastery.
func SuggestNextCharacter(lessonOrder []LessonCharacter, masteryByCharacter map[string]Mastery) (characterID string, glyph string, reasonCode string, masteryState string) {
	pick := func(wantState, reason string) (string, string, string, string) {
		for _, lc := range lessonOrder {
			m, ok := masteryByCharacter[lc.CharacterID]
			if !ok {
				m = DeriveMastery(nil)
			}
			if m.State == wantState {
				return lc.CharacterID, lc.Glyph, reason, m.State
			}
		}
		return "", "", "", ""
	}

	if id, g, r, s := pick(MasteryStateNotStarted, NextReasonFirstNotStarted); id != "" {
		return id, g, r, s
	}
	if id, g, r, s := pick(MasteryStateLearning, NextReasonContinueLearning); id != "" {
		return id, g, r, s
	}
	if id, g, r, s := pick(MasteryStatePassedOnce, NextReasonEncourageSteady); id != "" {
		return id, g, r, s
	}
	return "", "", NextReasonAllSteady, MasteryStateSteady
}
