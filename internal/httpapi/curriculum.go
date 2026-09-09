package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/deliium/drawing-board/internal/learn"
	"github.com/gorilla/mux"
)

type lessonExampleResponse struct {
	Word         string `json:"word"`
	Romanization string `json:"romanization"`
	MeaningEn    string `json:"meaningEn"`
}

type lessonCharacterResponse struct {
	ID            string                 `json:"id"`
	Glyph         string                 `json:"glyph"`
	Romanization  string                 `json:"romanization"`
	StrokeCount   int                    `json:"strokeCount"`
	Pronunciation json.RawMessage        `json:"pronunciation"`
	DescriptionEn string                 `json:"descriptionEn"`
	Example       lessonExampleResponse  `json:"example"`
	SortKey       int                    `json:"sortKey"`
	Position      int                    `json:"position"`
}

type lessonResponse struct {
	ID             string                    `json:"id"`
	Code           string                    `json:"code"`
	Title          string                    `json:"title"`
	SetID          string                    `json:"setId"`
	ContentVersion string                    `json:"contentVersion"`
	Characters     []lessonCharacterResponse `json:"characters"`
}

type progressItemResponse struct {
	CharacterID   string  `json:"characterId"`
	Status        string  `json:"status"`
	AttemptCount  int     `json:"attemptCount"`
	PassCount     int     `json:"passCount"`
	LastAttemptID *int64  `json:"lastAttemptId,omitempty"`
	LastPassedAt  *string `json:"lastPassedAt,omitempty"`
	UpdatedAt     string  `json:"updatedAt"`
}

type progressListResponse struct {
	Items []progressItemResponse `json:"items"`
}

// GetLesson serves GET /api/lessons/{id} for published lessons only.
func (a *API) GetLesson(w http.ResponseWriter, r *http.Request) {
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

	lessonID := strings.TrimSpace(mux.Vars(r)["id"])
	if lessonID == "" {
		apiLog("WARN", "[httpapi.Lesson.Get] not_found empty id userID=%d", uid)
		writeAPIError(w, 404, "not_found", "lesson not found")
		return
	}

	lesson, err := ls.Lessons().GetPublished(r.Context(), lessonID)
	if err != nil {
		if errors.Is(err, learn.ErrNotFound) {
			apiLog("WARN", "[httpapi.Lesson.Get] not_found userID=%d lessonId=%s", uid, lessonID)
			writeAPIError(w, 404, "not_found", "lesson not found")
			return
		}
		apiLog("ERROR", "[httpapi.Lesson.Get] store lessonId=%s: %v", lessonID, err)
		writeAPIError(w, 500, "internal_error", "failed to load lesson")
		return
	}

	placements, err := ls.Lessons().ListCharacters(r.Context(), lesson.ID)
	if err != nil {
		apiLog("ERROR", "[httpapi.Lesson.Get] list characters lessonId=%s: %v", lesson.ID, err)
		writeAPIError(w, 500, "internal_error", "failed to load lesson characters")
		return
	}

	chars := make([]lessonCharacterResponse, 0, len(placements))
	ids := make([]string, 0, len(placements))
	contentVersion := ""
	for _, lc := range placements {
		ch, err := ls.Characters().Get(r.Context(), lc.CharacterID)
		if err != nil {
			if errors.Is(err, learn.ErrNotFound) {
				apiLog("WARN", "[httpapi.Lesson.Get] missing character lessonId=%s characterId=%s", lesson.ID, lc.CharacterID)
				continue
			}
			apiLog("ERROR", "[httpapi.Lesson.Get] character lessonId=%s characterId=%s: %v", lesson.ID, lc.CharacterID, err)
			writeAPIError(w, 500, "internal_error", "failed to load lesson characters")
			return
		}
		if contentVersion == "" {
			contentVersion = ch.ContentVersion
		}
		pron := decodePronunciationJSON(ch.PronunciationJSON)
		chars = append(chars, lessonCharacterResponse{
			ID:            ch.ID,
			Glyph:         ch.Glyph,
			Romanization:  ch.Romanization,
			StrokeCount:   ch.StrokeCount,
			Pronunciation: pron,
			DescriptionEn: ch.DescriptionEn,
			Example: lessonExampleResponse{
				Word:         ch.ExampleWord,
				Romanization: ch.ExampleRomanization,
				MeaningEn:    ch.ExampleMeaningEn,
			},
			SortKey:  ch.SortKey,
			Position: lc.Position,
		})
		ids = append(ids, ch.ID)
	}

	apiLog("INFO", "[httpapi.Lesson.Get] userID=%d lessonId=%s characterCount=%d", uid, lesson.ID, len(chars))
	apiLog("DEBUG", "[httpapi.Lesson.Get] characterIds=%s", strings.Join(ids, ","))
	writeJSON(w, 200, lessonResponse{
		ID:             lesson.ID,
		Code:           lesson.Code,
		Title:          lesson.Title,
		SetID:          lesson.SetID,
		ContentVersion: contentVersion,
		Characters:     chars,
	})
}

// ListProgress serves GET /api/progress with optional lessonId / setId filters.
func (a *API) ListProgress(w http.ResponseWriter, r *http.Request) {
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
	setID := strings.TrimSpace(r.URL.Query().Get("setId"))

	rows, err := ls.Progress().ListForUser(r.Context(), uid)
	if err != nil {
		apiLog("ERROR", "[httpapi.Progress.List] store userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to load progress")
		return
	}

	allow := map[string]struct{}{}
	if lessonID != "" {
		if _, err := ls.Lessons().GetPublished(r.Context(), lessonID); err != nil {
			if errors.Is(err, learn.ErrNotFound) {
				apiLog("WARN", "[httpapi.Progress.List] not_found lessonId=%s userID=%d", lessonID, uid)
				writeAPIError(w, 404, "not_found", "lesson not found")
				return
			}
			apiLog("ERROR", "[httpapi.Progress.List] lesson lookup lessonId=%s: %v", lessonID, err)
			writeAPIError(w, 500, "internal_error", "failed to load progress")
			return
		}
		placements, err := ls.Lessons().ListCharacters(r.Context(), lessonID)
		if err != nil {
			apiLog("ERROR", "[httpapi.Progress.List] list characters lessonId=%s: %v", lessonID, err)
			writeAPIError(w, 500, "internal_error", "failed to load progress")
			return
		}
		for _, lc := range placements {
			allow[lc.CharacterID] = struct{}{}
		}
	} else if setID != "" {
		chs, err := ls.Characters().ListBySet(r.Context(), setID)
		if err != nil {
			apiLog("ERROR", "[httpapi.Progress.List] list set setId=%s: %v", setID, err)
			writeAPIError(w, 500, "internal_error", "failed to load progress")
			return
		}
		for _, ch := range chs {
			allow[ch.ID] = struct{}{}
		}
	}

	items := make([]progressItemResponse, 0, len(rows))
	for _, p := range rows {
		if len(allow) > 0 {
			if _, ok := allow[p.CharacterID]; !ok {
				continue
			}
		}
		items = append(items, progressItemFromLearn(p))
	}

	apiLog("INFO", "[httpapi.Progress.List] userID=%d progressCount=%d lessonId=%s setId=%s",
		uid, len(items), lessonID, setID)
	writeJSON(w, 200, progressListResponse{Items: items})
}

func progressItemFromLearn(p learn.Progress) progressItemResponse {
	item := progressItemResponse{
		CharacterID:   p.CharacterID,
		Status:        p.Status,
		AttemptCount:  p.AttemptCount,
		PassCount:     p.PassCount,
		LastAttemptID: p.LastAttemptID,
		UpdatedAt:     rfc3339(p.UpdatedAt),
	}
	if p.LastPassedAt != nil {
		s := rfc3339(*p.LastPassedAt)
		item.LastPassedAt = &s
	}
	return item
}

func decodePronunciationJSON(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return json.RawMessage(`{}`)
	}
	if !json.Valid([]byte(raw)) {
		apiLog("WARN", "[httpapi.Lesson.Get] invalid pronunciation JSON; returning empty object")
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}
