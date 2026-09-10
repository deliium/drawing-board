package learn

import (
	"fmt"
	"testing"
	"time"
)

func ts(n int) time.Time {
	return time.Date(2026, 9, 1, 12, 0, n, 0, time.UTC)
}

func outcomes(passes ...bool) []AssessedOutcome {
	out := make([]AssessedOutcome, len(passes))
	for i, p := range passes {
		out[i] = AssessedOutcome{AttemptID: fmt.Sprintf("attempt-%d", i+1), Pass: p, AssessedAt: ts(i)}
	}
	return out
}

func TestDeriveMastery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		in       []AssessedOutcome
		state    string
		reason   string
		assessed int
		pass     int
		fail     int
		consec   int
		lastPass *bool
	}{
		{
			name:     "empty_not_started",
			in:       nil,
			state:    MasteryStateNotStarted,
			reason:   MasteryReasonNoAssessedAttempts,
			assessed: 0,
		},
		{
			name:     "all_fail_learning",
			in:       outcomes(false, false),
			state:    MasteryStateLearning,
			reason:   MasteryReasonNoPassesYet,
			assessed: 2,
			pass:     0,
			fail:     2,
			consec:   0,
			lastPass: boolPtr(false),
		},
		{
			name:     "one_pass_passed_once",
			in:       outcomes(true),
			state:    MasteryStatePassedOnce,
			reason:   MasteryReasonSinglePass,
			assessed: 1,
			pass:     1,
			fail:     0,
			consec:   1,
			lastPass: boolPtr(true),
		},
		{
			name:     "fail_then_pass_passed_once",
			in:       outcomes(false, true),
			state:    MasteryStatePassedOnce,
			reason:   MasteryReasonSinglePass,
			assessed: 2,
			pass:     1,
			fail:     1,
			consec:   1,
			lastPass: boolPtr(true),
		},
		{
			name:     "two_consecutive_steady",
			in:       outcomes(false, true, true),
			state:    MasteryStateSteady,
			reason:   MasteryReasonTwoConsecutivePasses,
			assessed: 3,
			pass:     2,
			fail:     1,
			consec:   2,
			lastPass: boolPtr(true),
		},
		{
			name:     "fail_after_passes_recent_miss",
			in:       outcomes(true, true, false),
			state:    MasteryStatePassedOnce,
			reason:   MasteryReasonRecentMiss,
			assessed: 3,
			pass:     2,
			fail:     1,
			consec:   0,
			lastPass: boolPtr(false),
		},
		{
			name:     "three_consecutive_still_steady",
			in:       outcomes(true, true, true),
			state:    MasteryStateSteady,
			reason:   MasteryReasonTwoConsecutivePasses,
			assessed: 3,
			pass:     3,
			fail:     0,
			consec:   3,
			lastPass: boolPtr(true),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DeriveMastery(tc.in)
			if got.State != tc.state || got.ReasonCode != tc.reason {
				t.Fatalf("state/reason=%s/%s want %s/%s", got.State, got.ReasonCode, tc.state, tc.reason)
			}
			if got.AssessedCount != tc.assessed || got.PassCount != tc.pass || got.FailCount != tc.fail {
				t.Fatalf("counts assessed=%d pass=%d fail=%d want %d/%d/%d",
					got.AssessedCount, got.PassCount, got.FailCount, tc.assessed, tc.pass, tc.fail)
			}
			if got.ConsecutivePassesEnding != tc.consec {
				t.Fatalf("consec=%d want %d", got.ConsecutivePassesEnding, tc.consec)
			}
			if tc.lastPass == nil {
				if got.LastPass != nil {
					t.Fatalf("LastPass want nil got %v", *got.LastPass)
				}
			} else if got.LastPass == nil || *got.LastPass != *tc.lastPass {
				t.Fatalf("LastPass=%v want %v", got.LastPass, *tc.lastPass)
			}
		})
	}
}

func TestSuggestNextCharacter(t *testing.T) {
	t.Parallel()
	order := []LessonCharacter{
		{CharacterID: "hira:あ", Glyph: "あ", Position: 1},
		{CharacterID: "hira:い", Glyph: "い", Position: 2},
		{CharacterID: "hira:う", Glyph: "う", Position: 3},
	}

	t.Run("fresh_first_not_started", func(t *testing.T) {
		id, glyph, reason, state := SuggestNextCharacter(order, nil)
		if id != "hira:あ" || glyph != "あ" || reason != NextReasonFirstNotStarted || state != MasteryStateNotStarted {
			t.Fatalf("got %s %s %s %s", id, glyph, reason, state)
		}
	})

	t.Run("skip_steady_to_learning", func(t *testing.T) {
		m := map[string]Mastery{
			"hira:あ": DeriveMastery(outcomes(true, true)),
			"hira:い": DeriveMastery(outcomes(false)),
			"hira:う": DeriveMastery(outcomes(true, true)),
		}
		id, _, reason, state := SuggestNextCharacter(order, m)
		if id != "hira:い" || reason != NextReasonContinueLearning || state != MasteryStateLearning {
			t.Fatalf("got %s %s %s", id, reason, state)
		}
	})

	t.Run("encourage_steady", func(t *testing.T) {
		m := map[string]Mastery{
			"hira:あ": DeriveMastery(outcomes(true, true)),
			"hira:い": DeriveMastery(outcomes(true, true)),
			"hira:う": DeriveMastery(outcomes(true)),
		}
		id, glyph, reason, state := SuggestNextCharacter(order, m)
		if id != "hira:う" || glyph != "う" || reason != NextReasonEncourageSteady || state != MasteryStatePassedOnce {
			t.Fatalf("got %s %s %s %s", id, glyph, reason, state)
		}
	})

	t.Run("all_steady", func(t *testing.T) {
		m := map[string]Mastery{
			"hira:あ": DeriveMastery(outcomes(true, true)),
			"hira:い": DeriveMastery(outcomes(true, true)),
			"hira:う": DeriveMastery(outcomes(true, true)),
		}
		id, _, reason, _ := SuggestNextCharacter(order, m)
		if id != "" || reason != NextReasonAllSteady {
			t.Fatalf("got id=%q reason=%s", id, reason)
		}
	})
}

func boolPtr(v bool) *bool { return &v }
