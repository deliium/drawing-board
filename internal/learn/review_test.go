package learn

import (
	"testing"
	"time"
)

func TestApplyReviewOutcome(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		box     int
		pass    bool
		wantBox int
		wantDue time.Time
	}{
		{"first_pass_0_to_1", 0, true, 1, now.Add(24 * time.Hour)},
		{"first_fail_stays_0", 0, false, 0, now},
		{"pass_1_to_2", 1, true, 2, now.Add(72 * time.Hour)},
		{"pass_2_to_3", 2, true, 3, now.Add(168 * time.Hour)},
		{"pass_3_clamps", 3, true, 3, now.Add(168 * time.Hour)},
		{"fail_3_to_2", 3, false, 2, now.Add(72 * time.Hour)},
		{"fail_1_to_0", 1, false, 0, now},
		{"clamp_negative_box", -2, true, 1, now.Add(24 * time.Hour)},
		{"clamp_high_box_fail", 9, false, 2, now.Add(72 * time.Hour)},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotBox, gotDue := ApplyReviewOutcome(tc.box, tc.pass, now)
			if gotBox != tc.wantBox {
				t.Fatalf("box=%d want %d", gotBox, tc.wantBox)
			}
			if !gotDue.Equal(tc.wantDue) {
				t.Fatalf("dueAt=%s want %s", gotDue.Format(time.RFC3339), tc.wantDue.Format(time.RFC3339))
			}
		})
	}
}

func TestIsDue(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	if IsDue(nil, now) {
		t.Fatal("nil due should not be due")
	}
	if !IsDue(&past, now) || !IsDue(&now, now) {
		t.Fatal("past and equal should be due")
	}
	if IsDue(&future, now) {
		t.Fatal("future should not be due")
	}
}

func TestSuggestNextWithReview(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	order := []LessonCharacter{
		{CharacterID: "hira:あ", Glyph: "あ", Position: 1},
		{CharacterID: "hira:い", Glyph: "い", Position: 2},
		{CharacterID: "hira:う", Glyph: "う", Position: 3},
	}
	steady := DeriveMastery(outcomes(true, true))
	learning := DeriveMastery(outcomes(false))
	once := DeriveMastery(outcomes(true))
	fresh := DeriveMastery(nil)

	t.Run("overdue_beats_not_started", func(t *testing.T) {
		past := now.Add(-48 * time.Hour)
		future := now.Add(24 * time.Hour)
		m := map[string]Mastery{
			"hira:あ": steady,
			"hira:い": fresh,
			"hira:う": fresh,
		}
		r := map[string]ReviewSnapshot{
			"hira:あ": {Box: 2, DueAt: &past, Scheduled: true},
			"hira:い": {},
			"hira:う": {Box: 1, DueAt: &future, Scheduled: true},
		}
		got := SuggestNextWithReview(order, m, r, now)
		if got.CharacterID != "hira:あ" || got.ReasonCode != NextReasonDueReview {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("earliest_due_wins", func(t *testing.T) {
		d1 := now.Add(-2 * time.Hour)
		d2 := now.Add(-10 * time.Hour)
		m := map[string]Mastery{"hira:あ": steady, "hira:い": steady, "hira:う": steady}
		r := map[string]ReviewSnapshot{
			"hira:あ": {Box: 2, DueAt: &d1, Scheduled: true},
			"hira:い": {Box: 1, DueAt: &d2, Scheduled: true},
		}
		got := SuggestNextWithReview(order, m, r, now)
		if got.CharacterID != "hira:い" || got.ReasonCode != NextReasonDueReview {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("introduce_when_nothing_due", func(t *testing.T) {
		future := now.Add(24 * time.Hour)
		m := map[string]Mastery{"hira:あ": steady, "hira:い": fresh, "hira:う": fresh}
		r := map[string]ReviewSnapshot{
			"hira:あ": {Box: 1, DueAt: &future, Scheduled: true},
		}
		got := SuggestNextWithReview(order, m, r, now)
		if got.CharacterID != "hira:い" || got.ReasonCode != NextReasonFirstNotStarted {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("continue_learning", func(t *testing.T) {
		m := map[string]Mastery{"hira:あ": steady, "hira:い": learning, "hira:う": fresh}
		// う is not_started — introduction wins before learning
		got := SuggestNextWithReview(order, m, nil, now)
		if got.CharacterID != "hira:う" || got.ReasonCode != NextReasonFirstNotStarted {
			t.Fatalf("got %+v", got)
		}
		m2 := map[string]Mastery{"hira:あ": steady, "hira:い": learning, "hira:う": steady}
		got2 := SuggestNextWithReview(order, m2, nil, now)
		if got2.CharacterID != "hira:い" || got2.ReasonCode != NextReasonContinueLearning {
			t.Fatalf("got %+v", got2)
		}
	})

	t.Run("encourage_steady", func(t *testing.T) {
		m := map[string]Mastery{"hira:あ": steady, "hira:い": steady, "hira:う": once}
		got := SuggestNextWithReview(order, m, nil, now)
		if got.CharacterID != "hira:う" || got.ReasonCode != NextReasonEncourageSteady {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("all_caught_up_with_next_due", func(t *testing.T) {
		dA := now.Add(7 * 24 * time.Hour)
		dI := now.Add(3 * 24 * time.Hour)
		m := map[string]Mastery{"hira:あ": steady, "hira:い": steady, "hira:う": steady}
		r := map[string]ReviewSnapshot{
			"hira:あ": {Box: 3, DueAt: &dA, Scheduled: true},
			"hira:い": {Box: 2, DueAt: &dI, Scheduled: true},
			"hira:う": {Box: 3, DueAt: &dA, Scheduled: true},
		}
		got := SuggestNextWithReview(order, m, r, now)
		if got.CharacterID != "" || got.ReasonCode != NextReasonAllCaughtUp {
			t.Fatalf("got %+v", got)
		}
		if got.NextDueCharacterID != "hira:い" || got.NextDueAt == nil || !got.NextDueAt.Equal(dI) {
			t.Fatalf("nextDue %+v", got)
		}
	})

	t.Run("advance_clock_makes_due", func(t *testing.T) {
		due := now.Add(24 * time.Hour)
		m := map[string]Mastery{"hira:あ": steady, "hira:い": steady, "hira:う": steady}
		r := map[string]ReviewSnapshot{
			"hira:あ": {Box: 1, DueAt: &due, Scheduled: true},
		}
		if got := SuggestNextWithReview(order, m, r, now); got.ReasonCode != NextReasonAllCaughtUp {
			t.Fatalf("before advance: %+v", got)
		}
		later := now.Add(25 * time.Hour)
		got := SuggestNextWithReview(order, m, r, later)
		if got.CharacterID != "hira:あ" || got.ReasonCode != NextReasonDueReview {
			t.Fatalf("after advance: %+v", got)
		}
	})
}

func TestIntervalDaysForBox(t *testing.T) {
	t.Parallel()
	if IntervalDaysForBox(0) != 0 || IntervalDaysForBox(1) != 1 || IntervalDaysForBox(2) != 3 || IntervalDaysForBox(3) != 7 {
		t.Fatalf("unexpected interval days")
	}
}
