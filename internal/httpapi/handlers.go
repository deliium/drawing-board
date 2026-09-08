package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/recognize"
)

type API struct {
	Auth             *auth.Service
	Store            *db.Store
	Recognizer       recognize.Recognizer
	RecognizeLimiter *limits.Limiter
}

type StrokePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Stroke struct {
	ID              int64         `json:"id"`
	Points          []StrokePoint `json:"points"`
	Color           string        `json:"color"`
	Width           int           `json:"width"`
	ClientID        string        `json:"clientId"`
	StartedAtUnixMs int64         `json:"startedAtUnixMs"`
	OpID            string        `json:"opId,omitempty"`
}

type RecognizeRequest struct {
	TopN   *int `json:"topN"`
	Width  int  `json:"width"`
	Height int  `json:"height"`
}

type RecognizeResponse struct {
	Candidates []recognize.Candidate `json:"candidates"`
}

type apiErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, code int, errCode, message string) {
	writeJSON(w, code, apiErrorBody{Error: errCode, Message: message})
}

func apiLog(level, format string, args ...interface{}) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error":
		if level == "DEBUG" || level == "INFO" || level == "WARN" {
			return
		}
	case "warn":
		if level == "DEBUG" || level == "INFO" {
			return
		}
	case "info":
		if level == "DEBUG" {
			return
		}
	}
	log.Printf(level+" "+format, args...)
}

func (a *API) recognizeLimiter() *limits.Limiter {
	if a.RecognizeLimiter != nil {
		return a.RecognizeLimiter
	}
	a.RecognizeLimiter = limits.NewLimiter(limits.RecognizeRatePerMin, limits.RecognizeBurst)
	return a.RecognizeLimiter
}

func (a *API) ListStrokes(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	apiLog("DEBUG", "[httpapi.ListStrokes] userID=%d", uid)
	rows, err := a.Store.ListStrokesByUser(uid)
	if err != nil {
		apiLog("ERROR", "[httpapi.ListStrokes] store userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to list strokes")
		return
	}
	out := make([]Stroke, 0, len(rows))
	for _, s := range rows {
		pts := make([]StrokePoint, 0, len(s.Points))
		for _, p := range s.Points {
			pts = append(pts, StrokePoint{X: p.X, Y: p.Y})
		}
		out = append(out, Stroke{
			ID:              s.ID,
			Points:          pts,
			Color:           s.Color,
			Width:           s.Width,
			ClientID:        "",
			StartedAtUnixMs: s.StartedAtUnixMs,
			OpID:            s.OpID,
		})
	}
	writeJSON(w, 200, out)
}

func (a *API) ClearStrokes(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	apiLog("DEBUG", "[httpapi.ClearStrokes] userID=%d", uid)
	before, listErr := a.Store.ListStrokesByUser(uid)
	if listErr != nil {
		apiLog("ERROR", "[httpapi.ClearStrokes] list-before userID=%d: %v", uid, listErr)
	}
	if err := a.Store.ClearStrokesByUser(uid); err != nil {
		apiLog("ERROR", "[httpapi.ClearStrokes] store userID=%d: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to clear strokes")
		return
	}
	count := 0
	if listErr == nil {
		count = len(before)
	}
	apiLog("INFO", "[httpapi.ClearStrokes] clear completed userID=%d count=%d", uid, count)
	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func (a *API) DeleteStroke(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, 400, "bad_id", "invalid stroke id")
		return
	}
	apiLog("DEBUG", "[httpapi.DeleteStroke] userID=%d id=%d", uid, id)
	if err := a.Store.DeleteStroke(uid, id); err != nil {
		apiLog("ERROR", "[httpapi.DeleteStroke] store userID=%d id=%d: %v", uid, id, err)
		writeAPIError(w, 500, "internal_error", "failed to delete stroke")
		return
	}
	apiLog("INFO", "[httpapi.DeleteStroke] delete success userID=%d id=%d", uid, id)
	writeJSON(w, 200, map[string]any{"ok": true, "id": id})
}

func (a *API) Recognize(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok {
		writeAPIError(w, 401, "unauthorized", "authentication required")
		return
	}
	if a.Recognizer == nil {
		metrics.Add("recognize_requests_total{result=error}", 1)
		writeAPIError(w, 503, "recognizer_unavailable", "recognizer unavailable")
		return
	}

	if !a.recognizeLimiter().Allow(uid) {
		metrics.Add("recognize_requests_total{result=reject}", 1)
		metrics.Add("recognize_reject_total{code=rate_limited}", 1)
		apiLog("WARN", "[httpapi.Recognize] userID=%d result=reject code=rate_limited", uid)
		writeAPIError(w, 429, "rate_limited", "too many recognition requests")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxRecognizeBodyBytes)
	var req RecognizeRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		code, msg := mapRecognizeDecodeError(err)
		metrics.Add("recognize_requests_total{result=reject}", 1)
		metrics.Add("recognize_reject_total{code="+code+"}", 1)
		apiLog("DEBUG", "[httpapi.Recognize] userID=%d result=reject code=%s", uid, code)
		writeAPIError(w, 400, code, msg)
		return
	}

	topN := 0
	present := req.TopN != nil
	if present {
		topN = *req.TopN
	}
	normalizedTopN, err := limits.NormalizeOrValidateTopN(topN, present)
	if err != nil {
		a.rejectRecognize(w, uid, limits.ErrorCode(err), limits.SafeMessage(err))
		return
	}
	if err := limits.CheckCanvas(req.Width, req.Height); err != nil {
		a.rejectRecognize(w, uid, limits.ErrorCode(err), limits.SafeMessage(err))
		return
	}

	strokes, err := a.Store.ListStrokesByUser(uid)
	if err != nil {
		metrics.Add("recognize_requests_total{result=error}", 1)
		apiLog("ERROR", "[httpapi.Recognize] userID=%d store: %v", uid, err)
		writeAPIError(w, 500, "internal_error", "failed to load strokes")
		return
	}

	totalPoints := 0
	for _, s := range strokes {
		n := len(s.Points)
		totalPoints += n
		if n > limits.MaxPointsPerStroke {
			a.rejectRecognize(w, uid, "too_many_points", limits.SafeMessage(limits.ErrTooManyPoints))
			return
		}
		for _, p := range s.Points {
			if err := limits.ValidateCoord(p.X, p.Y); err != nil {
				a.rejectRecognize(w, uid, "invalid_stroke_data", limits.SafeMessage(limits.ErrInvalidStrokeData))
				return
			}
		}
	}
	if err := limits.ValidateStrokeSet(len(strokes), totalPoints); err != nil {
		a.rejectRecognize(w, uid, limits.ErrorCode(err), limits.SafeMessage(err))
		return
	}

	rs := make([]recognize.Stroke, 0, len(strokes))
	for _, s := range strokes {
		ps := make([]recognize.Point, 0, len(s.Points))
		for _, p := range s.Points {
			ps = append(ps, recognize.Point{X: p.X, Y: p.Y})
		}
		rs = append(rs, recognize.Stroke{Points: ps})
	}

	cands, err := a.Recognizer.Recognize(rs, req.Width, req.Height, normalizedTopN)
	if err != nil {
		if errors.Is(err, limits.ErrInvalidDimensions) || errors.Is(err, limits.ErrInvalidTopN) {
			a.rejectRecognize(w, uid, limits.ErrorCode(err), limits.SafeMessage(err))
			return
		}
		metrics.Add("recognize_requests_total{result=error}", 1)
		apiLog("ERROR", "[httpapi.Recognize] userID=%d recognizer failed", uid)
		writeAPIError(w, 500, "internal_error", "recognition failed")
		return
	}

	metrics.Add("recognize_requests_total{result=ok}", 1)
	apiLog("INFO", "[httpapi.Recognize] userID=%d result=ok strokes=%d points=%d candidates=%d",
		uid, len(strokes), totalPoints, len(cands))
	writeJSON(w, 200, RecognizeResponse{Candidates: cands})
}

func (a *API) rejectRecognize(w http.ResponseWriter, uid int64, code, message string) {
	metrics.Add("recognize_requests_total{result=reject}", 1)
	metrics.Add("recognize_reject_total{code="+code+"}", 1)
	apiLog("DEBUG", "[httpapi.Recognize] userID=%d result=reject code=%s", uid, code)
	writeAPIError(w, 400, code, message)
}

func mapRecognizeDecodeError(err error) (code, message string) {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		return "payload_too_large", "request body too large"
	}
	if strings.Contains(err.Error(), "http: request body too large") {
		return "payload_too_large", "request body too large"
	}
	if errors.Is(err, io.EOF) {
		return "bad_json", "invalid JSON body"
	}
	return "bad_json", "invalid JSON body"
}
