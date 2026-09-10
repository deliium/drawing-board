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
	Get(ctx context.Context, userID, attemptID string) (*Attempt, error)
	GetByClientAttemptID(ctx context.Context, userID string, clientAttemptID string) (*Attempt, error)
	SubmitStrokes(ctx context.Context, userID, attemptID string, strokes []StrokeInput, w, h int) error
	// ListStrokes returns ordered attempt strokes + canvas size for submitted/assessed attempts (ownership-checked).
	ListStrokes(ctx context.Context, userID, attemptID string) (strokes []StrokeInput, width, height int, err error)
	MarkAssessed(ctx context.Context, userID, attemptID string) error
	Abandon(ctx context.Context, userID, attemptID string) error
	// List returns paginated personal attempt history (metadata + assessment summary; no stroke points).
	List(ctx context.Context, userID string, filter AttemptListFilter) (AttemptListResult, error)
	// ClearPracticeData deletes this user's practice attempts (CASCADE strokes/assessments) and progress rows.
	// Does not delete free-board strokes or the account.
	ClearPracticeData(ctx context.Context, userID string) (ClearPracticeDataResult, error)
}

// AssessmentRepo persists assessment results (with progress) and reads them back.
type AssessmentRepo interface {
	// SaveResult writes assessment + feedback and updates attempt + progress in one store tx.
	SaveResult(ctx context.Context, userID string, in SaveAssessment) (AssessmentResult, error)
	GetByAttempt(ctx context.Context, userID, attemptID string) (*AssessmentResult, error)
}

// ProgressRepo reads per-user character progress.
type ProgressRepo interface {
	Get(ctx context.Context, userID string, characterID string) (*Progress, error)
	ListForUser(ctx context.Context, userID string) ([]Progress, error)
	// ListAssessedOutcomes returns recent assessed pass/fail timelines per character (ASC within each id).
	// limitPerCharacter caps the newest window used for mastery derivation (≤0 → MasteryOutcomeWindow).
	ListAssessedOutcomes(ctx context.Context, userID string, characterIDs []string, limitPerCharacter int) (map[string][]AssessedOutcome, error)
}
