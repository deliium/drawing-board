package learn

import (
	"sort"
	"time"
)

// Leitner-style personal review schedule (engineering heuristic for hiragana5).
// Not SM-2/FSRS, not calibrated SRS science — fixed boxes + wall-clock intervals
// from assessed pass/fail only. No streaks or notifications.

const (
	ReviewBoxMin = 0
	ReviewBoxMax = 3
)

// Next suggestion reason codes for schedule-aware practice order.
const (
	NextReasonDueReview   = "due_review"
	NextReasonAllCaughtUp = "all_caught_up"
	// NextReasonFirstNotStarted / ContinueLearning / EncourageSteady reused from mastery.go.
)

// ReviewIntervals lists wall-clock duration after landing in each box.
var ReviewIntervals = [...]time.Duration{
	0,               // box 0 — due immediately
	24 * time.Hour,  // box 1 — 1 day
	72 * time.Hour,  // box 2 — 3 days
	168 * time.Hour, // box 3 — 7 days
}

// IntervalForBox returns the wall-clock interval for a box (clamped to 0–BoxMax).
func IntervalForBox(box int) time.Duration {
	if box < ReviewBoxMin {
		box = ReviewBoxMin
	}
	if box > ReviewBoxMax {
		box = ReviewBoxMax
	}
	return ReviewIntervals[box]
}

// IntervalDaysForBox is the informational day count for API/UI (0 for box 0).
func IntervalDaysForBox(box int) int {
	d := IntervalForBox(box)
	if d <= 0 {
		return 0
	}
	return int(d / (24 * time.Hour))
}

// ApplyReviewOutcome promotes/demotes the Leitner box from an assessed pass/fail
// and returns the new due_at (now + interval for the landing box).
// Fail demotes by one (forgiving), never resets to zero unless already at 0/1.
func ApplyReviewOutcome(box int, pass bool, now time.Time) (newBox int, dueAt time.Time) {
	if box < ReviewBoxMin {
		box = ReviewBoxMin
	}
	if box > ReviewBoxMax {
		box = ReviewBoxMax
	}
	now = now.UTC()
	if pass {
		newBox = box + 1
		if newBox > ReviewBoxMax {
			newBox = ReviewBoxMax
		}
	} else {
		newBox = box - 1
		if newBox < ReviewBoxMin {
			newBox = ReviewBoxMin
		}
	}
	dueAt = now.Add(IntervalForBox(newBox))
	return newBox, dueAt
}

// IsDue reports whether dueAt is non-nil and <= now (UTC comparison).
func IsDue(dueAt *time.Time, now time.Time) bool {
	if dueAt == nil {
		return false
	}
	return !dueAt.After(now.UTC())
}

// ReviewSnapshot is durable schedule state for one character (from progress row).
type ReviewSnapshot struct {
	Box            int
	DueAt          *time.Time
	LastReviewedAt *time.Time
	Scheduled      bool // true when due_at was ever set (character is in the review queue)
}

// NextSuggestion is the schedule-aware next-character pick (plus optional future due hint).
type NextSuggestion struct {
	CharacterID        string
	Glyph              string
	ReasonCode         string
	MasteryState       string
	DueAt              *time.Time
	ReviewBox          *int
	NextDueAt          *time.Time
	NextDueCharacterID string
}

// SuggestNextWithReview picks the next practice character using due reviews before
// introduction/learning. Mastery remains the practice summary; the schedule is when.
//
// Priority: due_review → first_not_started → continue_learning → encourage_steady → all_caught_up.
func SuggestNextWithReview(
	lessonOrder []LessonCharacter,
	masteryByCharacter map[string]Mastery,
	reviewByCharacter map[string]ReviewSnapshot,
	now time.Time,
) NextSuggestion {
	now = now.UTC()
	masteryOf := func(id string) Mastery {
		if m, ok := masteryByCharacter[id]; ok {
			return m
		}
		return DeriveMastery(nil)
	}
	reviewOf := func(id string) ReviewSnapshot {
		if r, ok := reviewByCharacter[id]; ok {
			return r
		}
		return ReviewSnapshot{}
	}

	type cand struct {
		lc      LessonCharacter
		mastery Mastery
		review  ReviewSnapshot
		dueAt   time.Time
		hasDue  bool
	}
	cands := make([]cand, 0, len(lessonOrder))
	for _, lc := range lessonOrder {
		r := reviewOf(lc.CharacterID)
		c := cand{lc: lc, mastery: masteryOf(lc.CharacterID), review: r}
		if r.DueAt != nil {
			c.dueAt = r.DueAt.UTC()
			c.hasDue = true
		}
		cands = append(cands, c)
	}

	pick := func(c cand, reason string) NextSuggestion {
		box := c.review.Box
		out := NextSuggestion{
			CharacterID:  c.lc.CharacterID,
			Glyph:        c.lc.Glyph,
			ReasonCode:   reason,
			MasteryState: c.mastery.State,
			ReviewBox:    &box,
		}
		if c.hasDue {
			t := c.dueAt
			out.DueAt = &t
		}
		return out
	}

	// 1) Due review — earliest due_at, then lesson position.
	var due []cand
	for _, c := range cands {
		if c.hasDue && IsDue(&c.dueAt, now) {
			due = append(due, c)
		}
	}
	if len(due) > 0 {
		sort.SliceStable(due, func(i, j int) bool {
			if !due[i].dueAt.Equal(due[j].dueAt) {
				return due[i].dueAt.Before(due[j].dueAt)
			}
			return due[i].lc.Position < due[j].lc.Position
		})
		return pick(due[0], NextReasonDueReview)
	}

	// 2) Introduce new — first not_started.
	for _, c := range cands {
		if c.mastery.State == MasteryStateNotStarted {
			out := pick(c, NextReasonFirstNotStarted)
			out.ReviewBox = nil // unscheduled introduction
			out.DueAt = nil
			return out
		}
	}

	// 3) Continue learning — prefer earliest scheduled due, else lesson order.
	var learning []cand
	for _, c := range cands {
		if c.mastery.State == MasteryStateLearning {
			learning = append(learning, c)
		}
	}
	if len(learning) > 0 {
		sort.SliceStable(learning, func(i, j int) bool {
			if learning[i].hasDue != learning[j].hasDue {
				return learning[i].hasDue
			}
			if learning[i].hasDue && learning[j].hasDue && !learning[i].dueAt.Equal(learning[j].dueAt) {
				return learning[i].dueAt.Before(learning[j].dueAt)
			}
			return learning[i].lc.Position < learning[j].lc.Position
		})
		return pick(learning[0], NextReasonContinueLearning)
	}

	// 4) Encourage steadiness — passed_once, prefer soonest future due.
	var once []cand
	for _, c := range cands {
		if c.mastery.State == MasteryStatePassedOnce {
			once = append(once, c)
		}
	}
	if len(once) > 0 {
		sort.SliceStable(once, func(i, j int) bool {
			if once[i].hasDue != once[j].hasDue {
				return once[i].hasDue
			}
			if once[i].hasDue && once[j].hasDue && !once[i].dueAt.Equal(once[j].dueAt) {
				return once[i].dueAt.Before(once[j].dueAt)
			}
			return once[i].lc.Position < once[j].lc.Position
		})
		return pick(once[0], NextReasonEncourageSteady)
	}

	// 5) Caught up — nothing due; surface next future review when present.
	out := NextSuggestion{
		ReasonCode:   NextReasonAllCaughtUp,
		MasteryState: MasteryStateSteady,
	}
	var future []cand
	for _, c := range cands {
		if c.hasDue && c.dueAt.After(now) {
			future = append(future, c)
		}
	}
	if len(future) > 0 {
		sort.SliceStable(future, func(i, j int) bool {
			if !future[i].dueAt.Equal(future[j].dueAt) {
				return future[i].dueAt.Before(future[j].dueAt)
			}
			return future[i].lc.Position < future[j].lc.Position
		})
		t := future[0].dueAt
		out.NextDueAt = &t
		out.NextDueCharacterID = future[0].lc.CharacterID
	}
	return out
}

// CountDue returns how many review snapshots are due at now.
func CountDue(reviews map[string]ReviewSnapshot, now time.Time) int {
	n := 0
	for _, r := range reviews {
		if IsDue(r.DueAt, now) {
			n++
		}
	}
	return n
}
