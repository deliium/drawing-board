package learn

import "time"

// AttemptListFilter scopes paginated personal attempt history.
type AttemptListFilter struct {
	LessonID    string
	CharacterID string
	// Statuses is non-empty; only assessed and/or abandoned are valid for list APIs.
	Statuses []string
	Limit    int
	// AfterStartedAt + AfterID form an opaque DESC cursor (exclusive): rows older than this pair.
	AfterStartedAt *time.Time
	AfterID        *string
}

// AttemptHistoryItem is a list-row summary — never includes stroke points.
type AttemptHistoryItem struct {
	ID          string
	CharacterID string
	Glyph       string
	LessonID    string
	Status      string
	StartedAt   time.Time
	AssessedAt  *time.Time
	Pass        *bool
	Score       *float64
	ScoreKind   string
	Feedback    []FeedbackItem
}

// AttemptListResult is a page of history items.
type AttemptListResult struct {
	Items         []AttemptHistoryItem
	NextStartedAt *time.Time
	NextID        *string
	HasNext       bool
}

// ClearPracticeDataResult reports how many personal practice rows were removed.
type ClearPracticeDataResult struct {
	AttemptsDeleted     int64
	ProgressRowsCleared int64
}
