package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deliium/drawing-board/internal/learn"
)

type masteryResponse struct {
	State                   string  `json:"state"`
	ReasonCode              string  `json:"reasonCode"`
	AssessedCount           int     `json:"assessedCount"`
	PassCount               int     `json:"passCount"`
	FailCount               int     `json:"failCount"`
	ConsecutivePassesEnding int     `json:"consecutivePassesEnding"`
	LastPass                *bool   `json:"lastPass,omitempty"`
	LastAssessedAt          *string `json:"lastAssessedAt,omitempty"`
}

type progressNextResponse struct {
	LessonID           string  `json:"lessonId"`
	CharacterID        *string `json:"characterId"`
	Glyph              *string `json:"glyph,omitempty"`
	ReasonCode         string  `json:"reasonCode"`
	MasteryState       string  `json:"masteryState,omitempty"`
	DueAt              *string `json:"dueAt,omitempty"`
	ReviewBox          *int    `json:"reviewBox,omitempty"`
	NextDueAt          *string `json:"nextDueAt,omitempty"`
	NextDueCharacterID *string `json:"nextDueCharacterId,omitempty"`
}

type clearPracticeDataResponse struct {
	AttemptsDeleted     int64 `json:"attemptsDeleted"`
	ProgressRowsCleared int64 `json:"progressRowsCleared"`
}

type attemptHistoryItemResponse struct {
	ID          int64                `json:"id"`
	CharacterID string               `json:"characterId"`
	Glyph       string               `json:"glyph"`
	LessonID    string               `json:"lessonId,omitempty"`
	Status      string               `json:"status"`
	StartedAt   string               `json:"startedAt"`
	AssessedAt  *string              `json:"assessedAt,omitempty"`
	Pass        *bool                `json:"pass,omitempty"`
	Score       *float64             `json:"score,omitempty"`
	ScoreKind   string               `json:"scoreKind,omitempty"`
	Feedback    []learn.FeedbackItem `json:"feedback,omitempty"`
}

type attemptHistoryListResponse struct {
	Items      []attemptHistoryItemResponse `json:"items"`
	NextCursor *string                      `json:"nextCursor,omitempty"`
	Limit      int                          `json:"limit"`
}

type attemptListCursorPayload struct {
	StartedAt string `json:"startedAt"`
	ID        int64  `json:"id"`
}

func encodeAttemptCursor(startedAt time.Time, id int64) string {
	raw, _ := json.Marshal(attemptListCursorPayload{StartedAt: startedAt.UTC().Format(time.RFC3339Nano), ID: id})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeAttemptCursor(raw string) (time.Time, int64, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, 0, err
	}
	var p attemptListCursorPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return time.Time{}, 0, err
	}
	t, err := time.Parse(time.RFC3339Nano, p.StartedAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, p.StartedAt)
	}
	if err != nil || p.ID <= 0 {
		return time.Time{}, 0, errors.New("invalid cursor")
	}
	return t.UTC(), p.ID, nil
}

func masteryFromLearn(m learn.Mastery) masteryResponse {
	out := masteryResponse{
		State:                   m.State,
		ReasonCode:              m.ReasonCode,
		AssessedCount:           m.AssessedCount,
		PassCount:               m.PassCount,
		FailCount:               m.FailCount,
		ConsecutivePassesEnding: m.ConsecutivePassesEnding,
		LastPass:                m.LastPass,
	}
	if m.LastAssessedAt != nil {
		s := rfc3339(*m.LastAssessedAt)
		out.LastAssessedAt = &s
	}
	return out
}

// ListAttempts serves GET /api/attempts — paginated personal history (no stroke points).
func (a *API) ListAttempts(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	limit := 20
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 50 {
			apiLog("WARN", "[httpapi.Attempts.List] bad limit=%q userID=%d", raw, uid)
			writeAPIError(w, 400, "invalid_input", "limit must be 1–50")
			return
		}
		limit = n
	}

	statusRaw := strings.TrimSpace(q.Get("status"))
	var statuses []string
	if statusRaw == "" {
		statuses = []string{learn.AttemptStatusAssessed}
	} else {
		for _, part := range strings.Split(statusRaw, ",") {
			st := strings.TrimSpace(part)
			if st == "" {
				continue
			}
			if st != learn.AttemptStatusAssessed && st != learn.AttemptStatusAbandoned {
				apiLog("WARN", "[httpapi.Attempts.List] bad status=%q userID=%d", statusRaw, uid)
				writeAPIError(w, 400, "invalid_input", "status may only include assessed and/or abandoned")
				return
			}
			statuses = append(statuses, st)
		}
		if len(statuses) == 0 {
			writeAPIError(w, 400, "invalid_input", "status may only include assessed and/or abandoned")
			return
		}
	}

	filter := learn.AttemptListFilter{
		LessonID:    strings.TrimSpace(q.Get("lessonId")),
		CharacterID: strings.TrimSpace(q.Get("characterId")),
		Statuses:    statuses,
		Limit:       limit,
	}
	if cur := strings.TrimSpace(q.Get("cursor")); cur != "" {
		startedAt, id, err := decodeAttemptCursor(cur)
		if err != nil {
			apiLog("WARN", "[httpapi.Attempts.List] bad cursor userID=%d", uid)
			writeAPIError(w, 400, "invalid_input", "invalid cursor")
			return
		}
		filter.AfterStartedAt = &startedAt
		filter.AfterID = &id
	}

	if filter.LessonID != "" {
		if _, err := ls.Lessons().GetPublished(r.Context(), filter.LessonID); err != nil {
			if errors.Is(err, learn.ErrNotFound) {
				writeAPIError(w, 404, "not_found", "lesson not found")
				return
			}
			apiLog("ERROR", "[httpapi.Attempts.List] lesson lookup: %v", err)
			writeAPIError(w, 500, "internal_error", "failed to list attempts")
			return
		}
	}

	result, err := ls.Attempts().List(r.Context(), uid, filter)
	if err != nil {
		if errors.Is(err, learn.ErrInvalidInput) {
			writeAPIError(w, 400, "invalid_input", "invalid input")
			return
		}
		apiLog("ERROR", "[httpapi.Attempts.List] store userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to list attempts")
		return
	}

	items := make([]attemptHistoryItemResponse, 0, len(result.Items))
	for _, it := range result.Items {
		row := attemptHistoryItemResponse{
			ID:          it.ID,
			CharacterID: it.CharacterID,
			Glyph:       it.Glyph,
			LessonID:    it.LessonID,
			Status:      it.Status,
			StartedAt:   rfc3339(it.StartedAt),
			AssessedAt:  rfc3339Ptr(it.AssessedAt),
			Pass:        it.Pass,
			Score:       it.Score,
			ScoreKind:   it.ScoreKind,
		}
		if len(it.Feedback) > 0 {
			row.Feedback = it.Feedback
		}
		items = append(items, row)
	}

	resp := attemptHistoryListResponse{Items: items, Limit: limit}
	if result.HasNext && result.NextStartedAt != nil && result.NextID != nil {
		c := encodeAttemptCursor(*result.NextStartedAt, *result.NextID)
		resp.NextCursor = &c
	}

	apiLog("INFO", "[httpapi.Attempts.List] userID=%d count=%d lessonId=%s characterId=%s hasNext=%v",
		uid, len(items), filter.LessonID, filter.CharacterID, resp.NextCursor != nil)
	writeJSON(w, 200, resp)
}

// GetProgressNext serves GET /api/progress/next — humble next-character suggestion.
func (a *API) GetProgressNext(w http.ResponseWriter, r *http.Request) {
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

	lessonID := strings.TrimSpace(r.URL.Query().Get("lessonId"))
	if lessonID == "" {
		lessonID = "lesson:hiragana5"
	}
	if _, err := ls.Lessons().GetPublished(r.Context(), lessonID); err != nil {
		if errors.Is(err, learn.ErrNotFound) {
			writeAPIError(w, 404, "not_found", "lesson not found")
			return
		}
		apiLog("ERROR", "[httpapi.Progress.Next] lesson: %v", err)
		writeAPIError(w, 500, "internal_error", "failed to load suggestion")
		return
	}
	placements, err := ls.Lessons().ListCharacters(r.Context(), lessonID)
	if err != nil {
		apiLog("ERROR", "[httpapi.Progress.Next] placements: %v", err)
		writeAPIError(w, 500, "internal_error", "failed to load suggestion")
		return
	}
	ids := make([]string, 0, len(placements))
	for _, lc := range placements {
		ids = append(ids, lc.CharacterID)
	}
	outcomes, err := ls.Progress().ListAssessedOutcomes(r.Context(), uid, ids, learn.MasteryOutcomeWindow)
	if err != nil {
		apiLog("ERROR", "[httpapi.Progress.Next] outcomes userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to load suggestion")
		return
	}
	masteryBy := make(map[string]learn.Mastery, len(ids))
	for _, id := range ids {
		m := learn.DeriveMastery(outcomes[id])
		masteryBy[id] = m
		apiLog("DEBUG", "[learn.mastery] userID=%d characterID=%s state=%s reasonCode=%s assessedCount=%d",
			uid, id, m.State, m.ReasonCode, m.AssessedCount)
	}

	progressRows, err := ls.Progress().ListForUser(r.Context(), uid)
	if err != nil {
		apiLog("ERROR", "[httpapi.Progress.Next] progress userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to load suggestion")
		return
	}
	reviewBy := make(map[string]learn.ReviewSnapshot, len(progressRows))
	for _, p := range progressRows {
		snap := learn.ReviewSnapshot{Box: p.ReviewBox, DueAt: p.DueAt, LastReviewedAt: p.LastReviewedAt}
		if p.DueAt != nil {
			snap.Scheduled = true
		}
		reviewBy[p.CharacterID] = snap
	}

	now := ls.Now()
	sug := learn.SuggestNextWithReview(placements, masteryBy, reviewBy, now)
	dueCount := learn.CountDue(reviewBy, now)
	apiLog("DEBUG", "[learn.review.Suggest] userID=%d lessonId=%s reasonCode=%s characterId=%s dueCount=%d nextDueCharacterId=%s",
		uid, lessonID, sug.ReasonCode, sug.CharacterID, dueCount, sug.NextDueCharacterID)

	resp := progressNextResponse{
		LessonID:     lessonID,
		ReasonCode:   sug.ReasonCode,
		MasteryState: sug.MasteryState,
		ReviewBox:    sug.ReviewBox,
	}
	if sug.CharacterID != "" {
		id := sug.CharacterID
		g := sug.Glyph
		resp.CharacterID = &id
		resp.Glyph = &g
	}
	if sug.DueAt != nil {
		s := rfc3339(*sug.DueAt)
		resp.DueAt = &s
	}
	if sug.NextDueAt != nil {
		s := rfc3339(*sug.NextDueAt)
		resp.NextDueAt = &s
	}
	if sug.NextDueCharacterID != "" {
		id := sug.NextDueCharacterID
		resp.NextDueCharacterID = &id
	}

	charLog := ""
	if resp.CharacterID != nil {
		charLog = *resp.CharacterID
	}
	dueLog := ""
	if resp.DueAt != nil {
		dueLog = *resp.DueAt
	}
	apiLog("INFO", "[httpapi.Progress.Next] userID=%d lessonId=%s characterId=%s reasonCode=%s dueAt=%s reviewBox=%v",
		uid, lessonID, charLog, sug.ReasonCode, dueLog, sug.ReviewBox)
	writeJSON(w, 200, resp)
}

// ClearPracticeData serves DELETE /api/practice-data — wipe personal practice history (not free-board).
func (a *API) ClearPracticeData(w http.ResponseWriter, r *http.Request) {
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

	res, err := ls.Attempts().ClearPracticeData(r.Context(), uid)
	if err != nil {
		apiLog("ERROR", "[httpapi.PracticeData.Clear] userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to clear practice data")
		return
	}
	apiLog("INFO", "[httpapi.PracticeData.Clear] userID=%d attemptsDeleted=%d progressRowsCleared=%d",
		uid, res.AttemptsDeleted, res.ProgressRowsCleared)
	writeJSON(w, 200, clearPracticeDataResponse{
		AttemptsDeleted:     res.AttemptsDeleted,
		ProgressRowsCleared: res.ProgressRowsCleared,
	})
}
