package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/recognize"
)

type API struct {
	Auth  *auth.Service
	Store *db.Store
	Recognizer recognize.Recognizer
}

type StrokePoint struct { X float64 `json:"x"`; Y float64 `json:"y"` }

type Stroke struct {
	ID int64 `json:"id"`
	Points []StrokePoint `json:"points"`
	Color string `json:"color"`
	Width int `json:"width"`
	ClientID string `json:"clientId"`
	StartedAtUnixMs int64 `json:"startedAtUnixMs"`
}

type RecognizeRequest struct {
	TopN int `json:"topN"`
	Width int `json:"width"`
	Height int `json:"height"`
}

type RecognizeResponse struct {
	Candidates []recognize.Candidate `json:"candidates"`
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) ListStrokes(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	rows, err := a.Store.ListStrokesByUser(uid)
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	out := make([]Stroke, 0, len(rows))
	for _, s := range rows {
		pts := make([]StrokePoint, 0, len(s.Points))
		for _, p := range s.Points { pts = append(pts, StrokePoint{X:p.X, Y:p.Y}) }
		out = append(out, Stroke{ID: s.ID, Points: pts, Color: s.Color, Width: s.Width, ClientID: "", StartedAtUnixMs: s.StartedAtUnixMs})
	}
	writeJSON(w, 200, out)
}

func (a *API) ClearStrokes(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	if err := a.Store.ClearStrokesByUser(uid); err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	writeJSON(w, 200, map[string]string{"ok":"true"})
}

func (a *API) DeleteStroke(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 { writeJSON(w, 400, map[string]string{"error":"bad id"}); return }
	if err := a.Store.DeleteStroke(uid, id); err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	writeJSON(w, 200, map[string]any{"ok": true, "id": id})
}

func (a *API) Recognize(w http.ResponseWriter, r *http.Request) {
	uid, ok := a.Auth.UserIDFromRequest(r)
	if !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	if a.Recognizer == nil { writeJSON(w, 503, map[string]string{"error":"recognizer unavailable"}); return }
	var req RecognizeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	strokes, err := a.Store.ListStrokesByUser(uid)
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	
	// Debug logging
	fmt.Printf("Recognition request: analyzing %d strokes for user %d\n", len(strokes), uid)
	for i, s := range strokes {
		fmt.Printf("  Stroke %d: %d points\n", i, len(s.Points))
	}
	
	rs := make([]recognize.Stroke, 0, len(strokes))
	for _, s := range strokes {
		ps := make([]recognize.Point, 0, len(s.Points))
		for _, p := range s.Points { ps = append(ps, recognize.Point{X:p.X, Y:p.Y}) }
		rs = append(rs, recognize.Stroke{ Points: ps })
	}
	cands, err := a.Recognizer.Recognize(rs, req.Width, req.Height, req.TopN)
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	
	// Debug logging
	fmt.Printf("Recognition result: %d candidates\n", len(cands))
	for i, c := range cands {
		fmt.Printf("  %d: %s (%.2f)\n", i, c.Text, c.Score)
	}
	
	writeJSON(w, 200, RecognizeResponse{ Candidates: cands })
}
