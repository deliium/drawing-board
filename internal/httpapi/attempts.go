package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/learn"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
	"github.com/gorilla/mux"
)

type createAttemptRequest struct {
	CharacterID     string `json:"characterId"`
	LessonID        string `json:"lessonId"`
	ClientAttemptID string `json:"clientAttemptId"`
}

type attemptResponse struct {
	ID              int64   `json:"id"`
	CharacterID     string  `json:"characterId"`
	Glyph           string  `json:"glyph,omitempty"`
	LessonID        string  `json:"lessonId,omitempty"`
	Status          string  `json:"status"`
	ClientAttemptID string  `json:"clientAttemptId,omitempty"`
	StartedAt       string  `json:"startedAt"`
	SubmittedAt     *string `json:"submittedAt,omitempty"`
	CanvasWidth     *int    `json:"canvasWidth,omitempty"`
	CanvasHeight    *int    `json:"canvasHeight,omitempty"`
	StrokeCount     *int    `json:"strokeCount,omitempty"`
}

type submitAttemptRequest struct {
	Width   int                   `json:"width"`
	Height  int                   `json:"height"`
	Strokes []submitStrokeRequest `json:"strokes"`
}

type submitStrokeRequest struct {
	Color           string        `json:"color"`
	Width           int           `json:"width"`
	StartedAtUnixMs int64         `json:"startedAtUnixMs"`
	Points          []StrokePoint `json:"points"`
}

type submitAttemptResponse struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submittedAt"`
	StrokeCount int    `json:"strokeCount"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type assessmentResponse struct {
	AttemptID   int64                 `json:"attemptId"`
	CharacterID string                `json:"characterId"`
	Glyph       string                `json:"glyph,omitempty"`
	Status      string                `json:"status"`
	Pass        bool                  `json:"pass"`
	Score       float64               `json:"score"`
	ScoreKind   string                `json:"scoreKind"`
	Assessor    string                `json:"assessor"`
	SetID       string                `json:"setId"`
	Reasons     []string              `json:"reasons"`
	Feedback    []learn.FeedbackItem  `json:"feedback"`
	Candidates  []recognize.Candidate `json:"candidates,omitempty"`
}

type abandonResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func (a *API) learnStore() *db.LearnStore {
	if a.Learn != nil {
		return a.Learn
	}
	if a.Store == nil {
		return nil
	}
	a.Learn = db.NewLearnStore(a.Store)
	return a.Learn
}

func attemptMetric(op, result string) {
	metrics.Add("attempt_requests_total{op="+op+",result="+result+"}", 1)
}

func attemptRejectMetric(op, code string) {
	attemptMetric(op, "reject")
	metrics.Add("attempt_reject_total{op="+op+",code="+code+"}", 1)
}

func writeLearnError(w http.ResponseWriter, err error, fallbackMsg string) {
	switch {
	case errors.Is(err, learn.ErrNotFound):
		writeAPIError(w, 404, "not_found", "not found")
	case errors.Is(err, learn.ErrConflict):
		writeAPIError(w, 409, "conflict", "conflict")
	case errors.Is(err, learn.ErrInvalidStatus):
		writeAPIError(w, 409, "invalid_status", "invalid attempt status")
	case errors.Is(err, learn.ErrInvalidInput):
		writeAPIError(w, 400, "invalid_input", "invalid input")
	default:
		writeAPIError(w, 500, "internal_error", fallbackMsg)
	}
}

func rfc3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := rfc3339(*t)
	return &s
}

func parseAttemptID(r *http.Request) (int64, error) {
	raw := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, learn.ErrInvalidInput
	}
	return id, nil
}

func feedbackFromAssessment(items []recognize.FeedbackItem) []learn.FeedbackItem {
	if len(items) == 0 {
		return []learn.FeedbackItem{}
	}
	out := make([]learn.FeedbackItem, 0, len(items))
	for _, item := range items {
		out = append(out, learn.FeedbackItem{
			Rank:    item.Rank,
			Code:    item.Code,
			Message: item.Message,
		})
		if len(out) == 2 {
			break
		}
	}
	return out
}

func (a *API) CreateAttempt(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		attemptMetric("create", "error")
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxAPIJSONBodyBytes)
	var req createAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		code, msg := mapAttemptDecodeError(err)
		attemptRejectMetric("create", code)
		writeAPIError(w, 400, code, msg)
		return
	}
	req.CharacterID = strings.TrimSpace(req.CharacterID)
	req.LessonID = strings.TrimSpace(req.LessonID)
	req.ClientAttemptID = strings.TrimSpace(req.ClientAttemptID)
	apiLog("DEBUG", "[httpapi.Attempt.Create] userID=%d characterID=%s clientAttemptIDPresent=%t",
		uid, req.CharacterID, req.ClientAttemptID != "")

	if req.CharacterID == "" {
		attemptRejectMetric("create", "invalid_input")
		writeAPIError(w, 400, "invalid_input", "characterId required")
		return
	}
	if req.ClientAttemptID != "" {
		if err := limits.ValidateOpID(req.ClientAttemptID); err != nil {
			attemptRejectMetric("create", "invalid_input")
			writeAPIError(w, 400, "invalid_input", "invalid clientAttemptId")
			return
		}
	}

	ch, err := ls.Characters().Get(r.Context(), req.CharacterID)
	if err != nil {
		if errors.Is(err, learn.ErrNotFound) {
			attemptRejectMetric("create", "not_found")
			writeAPIError(w, 404, "not_found", "character not found")
			return
		}
		attemptMetric("create", "error")
		apiLog("ERROR", "[httpapi.Attempt.Create] character lookup userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to load character")
		return
	}
	if ch.Status != learn.CharacterStatusActive {
		attemptRejectMetric("create", "character_not_active")
		apiLog("WARN", "[httpapi.Attempt.Create] inactive character userID=%d characterID=%s", uid, ch.ID)
		writeAPIError(w, 400, "character_not_active", "character is not active")
		return
	}
	if ch.SetID != recognize.SetIDHiragana5 {
		attemptRejectMetric("create", "invalid_input")
		apiLog("WARN", "[httpapi.Attempt.Create] unsupported set userID=%d setID=%s", uid, ch.SetID)
		writeAPIError(w, 400, "invalid_input", "character set not supported for practice")
		return
	}

	if req.LessonID != "" {
		lesson, err := ls.Lessons().GetPublished(r.Context(), req.LessonID)
		if err != nil {
			if errors.Is(err, learn.ErrNotFound) {
				attemptRejectMetric("create", "not_found")
				writeAPIError(w, 404, "not_found", "lesson not found")
				return
			}
			attemptMetric("create", "error")
			writeAPIError(w, 500, "internal_error", "failed to load lesson")
			return
		}
		ordered, err := ls.Lessons().ListCharacters(r.Context(), lesson.ID)
		if err != nil {
			attemptMetric("create", "error")
			writeAPIError(w, 500, "internal_error", "failed to load lesson characters")
			return
		}
		found := false
		for _, lc := range ordered {
			if lc.CharacterID == ch.ID {
				found = true
				break
			}
		}
		if !found {
			attemptRejectMetric("create", "invalid_input")
			writeAPIError(w, 400, "invalid_input", "character not in lesson")
			return
		}
	}

	statusCode := 201
	if req.ClientAttemptID != "" {
		if existing, err := ls.Attempts().GetByClientAttemptID(r.Context(), uid, req.ClientAttemptID); err == nil {
			if existing.CharacterID != ch.ID || existing.LessonID != req.LessonID {
				attemptRejectMetric("create", "conflict")
				apiLog("WARN", "[httpapi.Attempt.Create] conflict userID=%d clientAttemptID=%s", uid, req.ClientAttemptID)
				writeAPIError(w, 409, "conflict", "clientAttemptId reused for different character/lesson")
				return
			}
			statusCode = 200
			attemptMetric("create", "ok")
			apiLog("INFO", "[httpapi.Attempt.Create] idempotent replay userID=%d attemptID=%d", uid, existing.ID)
			writeJSON(w, statusCode, attemptResponse{
				ID:              existing.ID,
				CharacterID:     existing.CharacterID,
				Glyph:           ch.Glyph,
				LessonID:        existing.LessonID,
				Status:          existing.Status,
				ClientAttemptID: existing.ClientAttemptID,
				StartedAt:       rfc3339(existing.StartedAt),
			})
			return
		} else if !errors.Is(err, learn.ErrNotFound) {
			attemptMetric("create", "error")
			apiLog("ERROR", "[httpapi.Attempt.Create] lookup clientAttemptID userID=%d: %v", uid, err)
			writeAPIError(w, 500, "internal_error", "failed to create attempt")
			return
		}
	}

	created, err := ls.Attempts().CreateDraft(r.Context(), learn.CreateDraft{
		UserID:          uid,
		CharacterID:     ch.ID,
		LessonID:        req.LessonID,
		ClientAttemptID: req.ClientAttemptID,
	})
	if err != nil {
		if errors.Is(err, learn.ErrConflict) {
			attemptRejectMetric("create", "conflict")
			apiLog("WARN", "[httpapi.Attempt.Create] conflict userID=%d clientAttemptID=%s", uid, req.ClientAttemptID)
			writeAPIError(w, 409, "conflict", "clientAttemptId reused for different character/lesson")
			return
		}
		attemptMetric("create", "error")
		apiLog("ERROR", "[httpapi.Attempt.Create] store userID=%d: %v", uid, err)
		writeLearnError(w, err, "failed to create attempt")
		return
	}

	attemptMetric("create", "ok")
	apiLog("INFO", "[httpapi.Attempt.Create] userID=%d attemptID=%d status=%s characterID=%s",
		uid, created.ID, created.Status, created.CharacterID)
	writeJSON(w, statusCode, attemptResponse{
		ID:              created.ID,
		CharacterID:     created.CharacterID,
		Glyph:           ch.Glyph,
		LessonID:        created.LessonID,
		Status:          created.Status,
		ClientAttemptID: created.ClientAttemptID,
		StartedAt:       rfc3339(created.StartedAt),
	})
}

func (a *API) GetAttempt(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}
	id, err := parseAttemptID(r)
	if err != nil {
		writeAPIError(w, 400, "invalid_input", "invalid attempt id")
		return
	}
	at, err := ls.Attempts().Get(r.Context(), uid, id)
	if err != nil {
		writeLearnError(w, err, "failed to load attempt")
		return
	}
	ch, err := ls.Characters().Get(r.Context(), at.CharacterID)
	glyph := ""
	if err == nil {
		glyph = ch.Glyph
	}
	resp := attemptResponse{
		ID:              at.ID,
		CharacterID:     at.CharacterID,
		Glyph:           glyph,
		LessonID:        at.LessonID,
		Status:          at.Status,
		ClientAttemptID: at.ClientAttemptID,
		StartedAt:       rfc3339(at.StartedAt),
		SubmittedAt:     rfc3339Ptr(at.SubmittedAt),
		CanvasWidth:     at.CanvasWidth,
		CanvasHeight:    at.CanvasHeight,
	}
	if at.Status == learn.AttemptStatusSubmitted || at.Status == learn.AttemptStatusAssessed {
		n, cerr := ls.CountAttemptStrokes(r.Context(), uid, id)
		if cerr == nil {
			resp.StrokeCount = &n
		}
	}
	writeJSON(w, 200, resp)
}

func (a *API) SubmitAttempt(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		attemptMetric("submit", "error")
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}
	id, err := parseAttemptID(r)
	if err != nil {
		attemptRejectMetric("submit", "invalid_input")
		writeAPIError(w, 400, "invalid_input", "invalid attempt id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxAPIJSONBodyBytes)
	var req submitAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		code, msg := mapAttemptDecodeError(err)
		attemptRejectMetric("submit", code)
		writeAPIError(w, 400, code, msg)
		return
	}
	if err := limits.CheckCanvas(req.Width, req.Height); err != nil {
		attemptRejectMetric("submit", limits.ErrorCode(err))
		writeAPIError(w, 400, limits.ErrorCode(err), limits.SafeMessage(err))
		return
	}
	totalPoints := 0
	for _, st := range req.Strokes {
		totalPoints += len(st.Points)
	}
	apiLog("DEBUG", "[httpapi.Attempt.Submit] userID=%d attemptID=%d strokeCount=%d pointCount=%d w=%d h=%d",
		uid, id, len(req.Strokes), totalPoints, req.Width, req.Height)
	if err := limits.ValidateStrokeSet(len(req.Strokes), totalPoints); err != nil {
		attemptRejectMetric("submit", limits.ErrorCode(err))
		writeAPIError(w, 400, limits.ErrorCode(err), limits.SafeMessage(err))
		return
	}

	strokes := make([]learn.StrokeInput, 0, len(req.Strokes))
	for _, st := range req.Strokes {
		color := st.Color
		if color == "" {
			color = "#000000"
		}
		width := st.Width
		if width == 0 {
			width = 2
		}
		if err := limits.ValidateStrokeMeta(width, color, ""); err != nil {
			attemptRejectMetric("submit", limits.ErrorCode(err))
			writeAPIError(w, 400, limits.ErrorCode(err), limits.SafeMessage(err))
			return
		}
		pts := make([]limits.FloatPoint, len(st.Points))
		learnPts := make([]learn.StrokePoint, len(st.Points))
		for i, p := range st.Points {
			pts[i] = limits.FloatPoint{X: p.X, Y: p.Y}
			learnPts[i] = learn.StrokePoint{X: p.X, Y: p.Y}
		}
		if err := limits.ValidateStrokePoints(pts); err != nil {
			attemptRejectMetric("submit", limits.ErrorCode(err))
			writeAPIError(w, 400, limits.ErrorCode(err), limits.SafeMessage(err))
			return
		}
		strokes = append(strokes, learn.StrokeInput{
			Color: color, Width: width, StartedAtUnixMs: st.StartedAtUnixMs, Points: learnPts,
		})
	}

	if err := ls.Attempts().SubmitStrokes(r.Context(), uid, id, strokes, req.Width, req.Height); err != nil {
		if errors.Is(err, learn.ErrInvalidStatus) {
			attemptRejectMetric("submit", "invalid_status")
			apiLog("WARN", "[httpapi.Attempt.Submit] invalid_status attemptID=%d", id)
			writeAPIError(w, 409, "invalid_status", "attempt is not draft")
			return
		}
		if errors.Is(err, learn.ErrNotFound) {
			attemptRejectMetric("submit", "not_found")
			writeAPIError(w, 404, "not_found", "not found")
			return
		}
		if errors.Is(err, learn.ErrInvalidInput) {
			attemptRejectMetric("submit", "invalid_input")
			writeAPIError(w, 400, "invalid_input", "invalid stroke payload")
			return
		}
		attemptMetric("submit", "error")
		apiLog("ERROR", "[httpapi.Attempt.Submit] store attemptID=%d: %v", id, err)
		writeAPIError(w, 500, "internal_error", "failed to submit attempt")
		return
	}
	at, err := ls.Attempts().Get(r.Context(), uid, id)
	if err != nil {
		attemptMetric("submit", "error")
		writeAPIError(w, 500, "internal_error", "failed to reload attempt")
		return
	}
	attemptMetric("submit", "ok")
	apiLog("INFO", "[httpapi.Attempt.Submit] userID=%d attemptID=%d status=submitted strokeCount=%d",
		uid, id, len(strokes))
	writeJSON(w, 200, submitAttemptResponse{
		ID:          id,
		Status:      learn.AttemptStatusSubmitted,
		SubmittedAt: rfc3339(*at.SubmittedAt),
		StrokeCount: len(strokes),
		Width:       req.Width,
		Height:      req.Height,
	})
}

func (a *API) AssessAttempt(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		attemptMetric("assess", "error")
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}
	if a.Assessor == nil {
		attemptMetric("assess", "error")
		writeAPIError(w, 503, "recognizer_unavailable", "assessor unavailable")
		return
	}
	id, err := parseAttemptID(r)
	if err != nil {
		attemptRejectMetric("assess", "invalid_input")
		writeAPIError(w, 400, "invalid_input", "invalid attempt id")
		return
	}
	if !a.recognizeLimiter().Allow(uid) {
		attemptRejectMetric("assess", "rate_limited")
		apiLog("WARN", "[httpapi.Attempt.Assess] rate_limited userID=%d attemptID=%d", uid, id)
		writeAPIError(w, 429, "rate_limited", "too many assessment requests")
		return
	}

	at, err := ls.Attempts().Get(r.Context(), uid, id)
	if err != nil {
		if errors.Is(err, learn.ErrNotFound) {
			attemptRejectMetric("assess", "not_found")
			writeAPIError(w, 404, "not_found", "not found")
			return
		}
		attemptMetric("assess", "error")
		writeAPIError(w, 500, "internal_error", "failed to load attempt")
		return
	}
	apiLog("DEBUG", "[httpapi.Attempt.Assess] userID=%d attemptID=%d status=%s", uid, id, at.Status)

	ch, err := ls.Characters().Get(r.Context(), at.CharacterID)
	if err != nil {
		attemptMetric("assess", "error")
		writeAPIError(w, 500, "internal_error", "failed to load character")
		return
	}

	if at.Status == learn.AttemptStatusAssessed {
		ar, err := ls.Assessments().GetByAttempt(r.Context(), uid, id)
		if err != nil {
			attemptMetric("assess", "error")
			writeLearnError(w, err, "failed to load assessment")
			return
		}
		attemptMetric("assess", "ok")
		apiLog("INFO", "[httpapi.Attempt.Assess] idempotent replay userID=%d attemptID=%d pass=%t", uid, id, ar.Pass)
		writeJSON(w, 200, assessmentResponse{
			AttemptID:   id,
			CharacterID: at.CharacterID,
			Glyph:       ch.Glyph,
			Status:      learn.AttemptStatusAssessed,
			Pass:        ar.Pass,
			Score:       ar.Score,
			ScoreKind:   ar.ScoreKind,
			Assessor:    ar.Assessor,
			SetID:       ar.SetID,
			Reasons:     ar.Reasons,
			Feedback:    ar.Feedback,
		})
		return
	}
	if at.Status != learn.AttemptStatusSubmitted {
		attemptRejectMetric("assess", "invalid_status")
		apiLog("WARN", "[httpapi.Attempt.Assess] invalid_status attemptID=%d status=%s", id, at.Status)
		writeAPIError(w, 409, "invalid_status", "attempt is not submitted")
		return
	}

	strokes, width, height, err := ls.Attempts().ListStrokes(r.Context(), uid, id)
	if err != nil {
		attemptMetric("assess", "error")
		apiLog("ERROR", "[httpapi.Attempt.Assess] list strokes attemptID=%d: %v", id, err)
		writeLearnError(w, err, "failed to load attempt strokes")
		return
	}
	rs := make([]recognize.Stroke, 0, len(strokes))
	for _, st := range strokes {
		ps := make([]recognize.Point, 0, len(st.Points))
		for _, p := range st.Points {
			ps = append(ps, recognize.Point{X: p.X, Y: p.Y})
		}
		rs = append(rs, recognize.Stroke{Points: ps})
	}

	assessment, err := a.Assessor.Assess(ch.Glyph, rs, width, height)
	if err != nil {
		if errors.Is(err, recognize.ErrUnsupportedTarget) {
			attemptRejectMetric("assess", "unsupported_target")
			writeAPIError(w, 400, "unsupported_target", "character not assessable")
			return
		}
		attemptMetric("assess", "error")
		apiLog("ERROR", "[httpapi.Attempt.Assess] assessor attemptID=%d: %v", id, err)
		writeAPIError(w, 500, "internal_error", "assessment failed")
		return
	}

	feedback := feedbackFromAssessment(assessment.Feedback)
	codes := make([]string, len(feedback))
	for i, f := range feedback {
		codes[i] = f.Code
	}
	apiLog("DEBUG", "[httpapi.Attempt.Assess] attemptID=%d pass=%t score=%.3f feedbackCodes=%v", id, assessment.Pass, assessment.Score, codes)
	ar, err := ls.Assessments().SaveResult(r.Context(), uid, learn.SaveAssessment{
		AttemptID: id,
		Pass:      assessment.Pass,
		Score:     assessment.Score,
		ScoreKind: recognize.ScoreKindMatch,
		Assessor:  learn.AssessorTargetCompare,
		SetID:     ch.SetID,
		Reasons:   assessment.Reasons,
		Feedback:  feedback,
	})
	if err != nil {
		attemptMetric("assess", "error")
		apiLog("ERROR", "[httpapi.Attempt.Assess] save result attemptID=%d: %v", id, err)
		writeLearnError(w, err, "failed to save assessment")
		return
	}

	attemptMetric("assess", "ok")
	apiLog("INFO", "[httpapi.Attempt.Assess] userID=%d attemptID=%d pass=%t score=%.3f scoreKind=%s feedbackCount=%d",
		uid, id, ar.Pass, ar.Score, ar.ScoreKind, len(ar.Feedback))
	apiLog("DEBUG", "[httpapi.Attempt.Assess] attemptID=%d status=assessed pass=%t score=%.3f", id, ar.Pass, ar.Score)
	writeJSON(w, 200, assessmentResponse{
		AttemptID:   id,
		CharacterID: at.CharacterID,
		Glyph:       ch.Glyph,
		Status:      learn.AttemptStatusAssessed,
		Pass:        ar.Pass,
		Score:       ar.Score,
		ScoreKind:   ar.ScoreKind,
		Assessor:    ar.Assessor,
		SetID:       ar.SetID,
		Reasons:     ar.Reasons,
		Feedback:    ar.Feedback,
		Candidates:  assessment.Candidates,
	})
}

func (a *API) GetAttemptAssessment(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}
	id, err := parseAttemptID(r)
	if err != nil {
		writeAPIError(w, 400, "invalid_input", "invalid attempt id")
		return
	}
	at, err := ls.Attempts().Get(r.Context(), uid, id)
	if err != nil {
		writeLearnError(w, err, "failed to load attempt")
		return
	}
	ar, err := ls.Assessments().GetByAttempt(r.Context(), uid, id)
	if err != nil {
		writeLearnError(w, err, "failed to load assessment")
		return
	}
	ch, err := ls.Characters().Get(r.Context(), at.CharacterID)
	glyph := ""
	if err == nil {
		glyph = ch.Glyph
	}
	writeJSON(w, 200, assessmentResponse{
		AttemptID:   id,
		CharacterID: at.CharacterID,
		Glyph:       glyph,
		Status:      learn.AttemptStatusAssessed,
		Pass:        ar.Pass,
		Score:       ar.Score,
		ScoreKind:   ar.ScoreKind,
		Assessor:    ar.Assessor,
		SetID:       ar.SetID,
		Reasons:     ar.Reasons,
		Feedback:    ar.Feedback,
	})
}

func (a *API) AbandonAttempt(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	ls := a.learnStore()
	if ls == nil {
		attemptMetric("abandon", "error")
		writeAPIError(w, 500, "internal_error", "learn store unavailable")
		return
	}
	id, err := parseAttemptID(r)
	if err != nil {
		attemptRejectMetric("abandon", "invalid_input")
		writeAPIError(w, 400, "invalid_input", "invalid attempt id")
		return
	}
	if err := ls.Attempts().Abandon(r.Context(), uid, id); err != nil {
		if errors.Is(err, learn.ErrInvalidStatus) {
			attemptRejectMetric("abandon", "invalid_status")
			apiLog("WARN", "[httpapi.Attempt.Abandon] invalid_status attemptID=%d", id)
			writeAPIError(w, 409, "invalid_status", "attempt is not draft")
			return
		}
		if errors.Is(err, learn.ErrNotFound) {
			attemptRejectMetric("abandon", "not_found")
			writeAPIError(w, 404, "not_found", "not found")
			return
		}
		attemptMetric("abandon", "error")
		apiLog("ERROR", "[httpapi.Attempt.Abandon] store attemptID=%d: %v", id, err)
		writeAPIError(w, 500, "internal_error", "failed to abandon attempt")
		return
	}
	attemptMetric("abandon", "ok")
	apiLog("INFO", "[httpapi.Attempt.Abandon] userID=%d attemptID=%d status=abandoned", uid, id)
	writeJSON(w, 200, abandonResponse{ID: id, Status: learn.AttemptStatusAbandoned})
}

func mapAttemptDecodeError(err error) (code, message string) {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		return "body_too_large", "request body too large"
	}
	if strings.Contains(err.Error(), "http: request body too large") {
		return "body_too_large", "request body too large"
	}
	if errors.Is(err, io.EOF) {
		return "bad_json", "invalid JSON body"
	}
	return "bad_json", "invalid JSON body"
}
