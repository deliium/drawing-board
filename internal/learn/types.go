// Package learn defines the durable learning-domain model and repository interfaces.
// Log prefix for implementations: [learn.*] (no stroke coordinates).
package learn

import "time"

// Character / lesson / attempt / progress status constants.
const (
	CharacterStatusActive  = "active"
	CharacterStatusRetired = "retired"

	LessonStatusDraft     = "draft"
	LessonStatusPublished = "published"

	AttemptStatusDraft     = "draft"
	AttemptStatusSubmitted = "submitted"
	AttemptStatusAssessed  = "assessed"
	AttemptStatusAbandoned = "abandoned"

	ProgressStatusUnseen     = "unseen"
	ProgressStatusSeen       = "seen"
	ProgressStatusPracticing = "practicing"
	ProgressStatusPassed     = "passed"

	AssessorTargetCompare = "target_compare"
)

// Character is a curriculum glyph (global, not per-user).
type Character struct {
	ID                  string
	SetID               string
	Glyph               string
	Romanization        string
	StrokeCount         int
	SortKey             int
	Status              string
	DescriptionEn       string
	DescriptionJa       string
	PronunciationJSON   string
	ExampleWord         string
	ExampleRomanization string
	ExampleMeaningEn    string
	ExampleMeaningJa    string
	ContentVersion      string
	TraceRef            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Lesson is an ordered practice unit over a character set.
type Lesson struct {
	ID        string
	Code      string
	Title     string
	TitleJa   string
	SetID     string
	SortOrder int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LessonCharacter is a character placement within a lesson.
type LessonCharacter struct {
	LessonID    string
	CharacterID string
	Glyph       string
	Position    int
}

// Attempt is a per-user practice submission lifecycle row.
type Attempt struct {
	ID              int64
	UserID          int64
	CharacterID     string
	LessonID        string // empty when null
	Status          string
	ClientAttemptID string // empty when null
	CanvasWidth     *int
	CanvasHeight    *int
	StartedAt       time.Time
	SubmittedAt     *time.Time
	AssessedAt      *time.Time
	AbandonedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StrokePoint is one CSS-logical point on an attempt stroke.
type StrokePoint struct {
	X float64
	Y float64
}

// StrokeInput is one stroke submitted with an attempt (no board opId).
type StrokeInput struct {
	Color           string
	Width           int
	StartedAtUnixMs int64
	Points          []StrokePoint
}

// FeedbackItem is one ranked machine-coded note on an assessment.
type FeedbackItem struct {
	Rank    int    `json:"rank"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AssessmentResult is the persisted target-comparison outcome for one attempt.
type AssessmentResult struct {
	ID         int64
	AttemptID  int64
	Pass       bool
	Score      float64
	ScoreKind  string
	Assessor   string
	SetID      string
	Reasons    []string
	Feedback   []FeedbackItem
	CreatedAt  time.Time
}

// Progress is per-user per-character practice state, including Leitner-style review fields.
type Progress struct {
	UserID           int64
	CharacterID      string
	Status           string
	AttemptCount     int
	PassCount        int
	LastAttemptID    *int64
	LastPassedAt     *time.Time
	ReviewBox        int
	DueAt            *time.Time
	LastReviewedAt   *time.Time
	UpdatedAt        time.Time
}

// CreateDraft is input for starting a practice attempt.
type CreateDraft struct {
	UserID          int64
	CharacterID     string
	LessonID        string // optional
	ClientAttemptID string // optional ≤36
}

// SaveAssessment is input for persisting assessment + feedback + progress in one tx.
type SaveAssessment struct {
	AttemptID int64
	Pass      bool
	Score     float64
	ScoreKind string
	Assessor  string
	SetID     string
	Reasons   []string
	Feedback  []FeedbackItem
}
