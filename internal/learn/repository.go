package learn

import "context"

// CharacterRepo reads global curriculum characters.
type CharacterRepo interface {
	ListBySet(ctx context.Context, setID string) ([]Character, error)
	Get(ctx context.Context, id string) (*Character, error)
}

// LessonRepo reads published lessons and their ordered characters.
type LessonRepo interface {
	GetPublished(ctx context.Context, id string) (*Lesson, error)
	ListCharacters(ctx context.Context, lessonID string) ([]LessonCharacter, error)
}

// AttemptRepo owns per-user practice attempt lifecycle writes.
type AttemptRepo interface {
	CreateDraft(ctx context.Context, in CreateDraft) (Attempt, error)
	Get(ctx context.Context, userID, attemptID int64) (*Attempt, error)
	SubmitStrokes(ctx context.Context, userID, attemptID int64, strokes []StrokeInput, w, h int) error
	MarkAssessed(ctx context.Context, userID, attemptID int64) error
	Abandon(ctx context.Context, userID, attemptID int64) error
}

// AssessmentRepo persists assessment results (with progress) and reads them back.
type AssessmentRepo interface {
	// SaveResult writes assessment + feedback and updates attempt + progress in one store tx.
	SaveResult(ctx context.Context, userID int64, in SaveAssessment) (AssessmentResult, error)
	GetByAttempt(ctx context.Context, userID, attemptID int64) (*AssessmentResult, error)
}

// ProgressRepo reads per-user character progress.
type ProgressRepo interface {
	Get(ctx context.Context, userID int64, characterID string) (*Progress, error)
	ListForUser(ctx context.Context, userID int64) ([]Progress, error)
}
