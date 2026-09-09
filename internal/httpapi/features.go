package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/deliium/drawing-board/internal/features"
	"github.com/deliium/drawing-board/internal/metrics"
)

const (
	metricFeatureDisabled = "practice_feature_disabled_total"
	metricAssessTotal     = "practice_attempt_assess_total"
	metricClearTotal      = "practice_clear_total"
)

// GetFeatures serves GET /api/features — authenticated kill-switch mirror for the SPA.
func (a *API) GetFeatures(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.Auth.UserIDFromRequest(r); !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	f := a.featureFlags()
	apiLog("DEBUG", "[httpapi.Features.Get] practice=%v progress=%v review=%v audio=%v",
		f.Practice, f.Progress, f.Review, f.Audio)
	writeJSON(w, 200, f)
}

func (a *API) featureFlags() features.Flags {
	if a.Features != nil {
		return *a.Features
	}
	// Safe default when tests omit Flags: all learning surfaces on.
	return features.Flags{Practice: true, Progress: true, Review: true, Audio: true}
}

func (a *API) RequireFeature(enabled bool, name string, next http.HandlerFunc) http.Handler {
	return a.requireFeature(enabled, name, next)
}

func (a *API) requireFeature(enabled bool, name string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			a.writeFeatureDisabled(w, name)
			return
		}
		next(w, r)
	})
}

func (a *API) writeFeatureDisabled(w http.ResponseWriter, name string) {
	metrics.Add(metricFeatureDisabled, 1)
	apiLog("WARN", "[httpapi] feature_disabled name=%s", name)
	// 404 (not 403) to avoid feature enumeration.
	writeAPIError(w, 404, "not_found", "not found")
}

func (a *API) stripAudioRef(pronunciation json.RawMessage) json.RawMessage {
	if a.featureFlags().Audio || len(pronunciation) == 0 {
		return pronunciation
	}
	var m map[string]interface{}
	if err := json.Unmarshal(pronunciation, &m); err != nil || m == nil {
		return pronunciation
	}
	if _, ok := m["audioRef"]; !ok {
		return pronunciation
	}
	delete(m, "audioRef")
	out, err := json.Marshal(m)
	if err != nil {
		return pronunciation
	}
	apiLog("DEBUG", "[httpapi.Lesson.Get] stripped audioRef feature_audio=off")
	return out
}
