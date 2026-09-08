package ws

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/metrics"
	"github.com/deliium/drawing-board/internal/security"
	"github.com/gorilla/websocket"
)

var (
	allowedOrigins []string
	upgrader       = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
	strokeLimiter = limits.NewLimiter(limits.StrokeIngestPerMin, limits.StrokeIngestBurst)
)

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		log.Printf("WARN [ws.CheckOrigin] rejected origin=")
		return false
	}
	if !security.OriginAllowed(allowedOrigins, origin) {
		log.Printf("WARN [ws.CheckOrigin] rejected origin=%s", origin)
		return false
	}
	log.Printf("DEBUG [ws.CheckOrigin] accepted origin=%s", origin)
	return true
}

// CheckOriginForTest exposes the WebSocket origin policy for unit tests.
func CheckOriginForTest(r *http.Request) bool {
	return checkOrigin(r)
}

// SetAllowedOriginsForTest replaces the allowlist used by CheckOrigin (tests only).
func SetAllowedOriginsForTest(origins []string) {
	allowedOrigins = append([]string(nil), origins...)
}

// SetStrokeLimiterForTest replaces the stroke ingest limiter (tests only).
func SetStrokeLimiterForTest(l *limits.Limiter) {
	strokeLimiter = l
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Stroke struct {
	ID              int64   `json:"id"`
	Points          []Point `json:"points"`
	Color           string  `json:"color"`
	Width           int     `json:"width"`
	ClientID        string  `json:"clientId"`
	StartedAtUnixMs int64   `json:"startedAtUnixMs"`
	OpID            string  `json:"opId,omitempty"`
}

type message struct {
	Type       string  `json:"type"`
	OpID       string  `json:"opId,omitempty"`
	BaseRev    *int64  `json:"baseRev,omitempty"`
	BoardRev   *int64  `json:"boardRev,omitempty"`
	Stroke     *Stroke `json:"stroke,omitempty"`
	Delete     *int64  `json:"delete,omitempty"`
	DeleteOpID string  `json:"deleteOpId,omitempty"`
	Clear      *bool   `json:"clear,omitempty"`
	StrokeID   *int64  `json:"strokeId,omitempty"`
	OK         *bool   `json:"ok,omitempty"`
	Error      string  `json:"error,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// writeMessageFn writes a text frame to a connection. Tests may override Hub.writeFn.
type writeMessageFn func(c *websocket.Conn, data []byte) error

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]int64
	Store   *db.Store
	Auth    *auth.Service
	writeFn writeMessageFn
}

func NewHub(store *db.Store, authSvc *auth.Service) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]int64),
		Store:   store,
		Auth:    authSvc,
		writeFn: defaultWriteMessage,
	}
}

func defaultWriteMessage(c *websocket.Conn, data []byte) error {
	c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.WriteMessage(websocket.TextMessage, data)
}

func (h *Hub) add(c *websocket.Conn, userID int64) {
	h.mu.Lock()
	h.clients[c] = userID
	h.mu.Unlock()
}

func (h *Hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) sendToUser(userID int64, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[ws.sendToUser] ERROR marshal userID=%d: %v", userID, err)
		return
	}
	write := h.writeFn
	if write == nil {
		write = defaultWriteMessage
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	recipients := 0
	for c, uid := range h.clients {
		if uid != userID {
			continue
		}
		recipients++
		if err := write(c, b); err != nil {
			if !isBenignNetErr(err) {
				log.Printf("[ws.sendToUser] ERROR write userID=%d: %v", userID, err)
			}
			c.Close()
			delete(h.clients, c)
		}
	}
	log.Printf("[ws.sendToUser] DEBUG userID=%d recipients=%d", userID, recipients)
}

func (h *Hub) sendToConn(conn *websocket.Conn, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[ws.sendToConn] ERROR marshal: %v", err)
		return
	}
	write := h.writeFn
	if write == nil {
		write = defaultWriteMessage
	}
	if err := write(conn, b); err != nil {
		if !isBenignNetErr(err) {
			log.Printf("[ws.sendToConn] ERROR write: %v", err)
		}
	}
}

func (h *Hub) sendAck(conn *websocket.Conn, opID string, ok bool, boardRev *int64, strokeID *int64, deleteID *int64, clear *bool, errCode, errMsg string) {
	ack := message{
		Type:    "ack",
		OpID:    opID,
		OK:      &ok,
		Error:   errCode,
		Message: errMsg,
	}
	if boardRev != nil {
		ack.BoardRev = boardRev
	}
	if strokeID != nil {
		ack.StrokeID = strokeID
	}
	if deleteID != nil {
		ack.Delete = deleteID
	}
	if clear != nil {
		ack.Clear = clear
	}
	h.sendToConn(conn, ack)
}

func (h *Hub) sendErrorToConn(conn *websocket.Conn, code, msg string) {
	b, err := json.Marshal(message{
		Type:    "error",
		Error:   code,
		Message: msg,
	})
	if err != nil {
		return
	}
	write := h.writeFn
	if write == nil {
		write = defaultWriteMessage
	}
	_ = write(conn, b)
}

var globalHub *Hub

// Init configures the global hub and WebSocket origin allowlist.
func Init(store *db.Store, authSvc *auth.Service, origins []string) {
	allowedOrigins = append([]string(nil), origins...)
	globalHub = NewHub(store, authSvc)
	log.Printf("INFO [ws.Init] origin_allowlist count=%d", len(allowedOrigins))
}

func Handle(w http.ResponseWriter, r *http.Request) {
	uid, ok := globalHub.Auth.UserIDFromRequest(r)
	if !ok {
		log.Printf("[ws.Handle] WARN missing userID remote=%s", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws.Handle] ERROR upgrade remote=%s userID=%d: %v", r.RemoteAddr, uid, err)
		return
	}
	log.Printf("[ws.Handle] DEBUG connect userID=%d remote=%s", uid, r.RemoteAddr)
	globalHub.add(conn, uid)
	defer func() {
		globalHub.remove(conn)
		conn.Close()
		log.Printf("[ws.Handle] DEBUG disconnect userID=%d remote=%s", uid, r.RemoteAddr)
	}()

	conn.SetReadLimit(int64(limits.MaxWSMessageBytes))
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	done := make(chan struct{})
	conn.SetCloseHandler(func(code int, text string) error {
		select {
		case <-done:
		default:
			close(done)
		}
		return nil
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					if !isBenignNetErr(err) {
						log.Printf("[ws.Handle] ERROR ping userID=%d: %v", uid, err)
					}
					_ = conn.Close()
					select {
					case <-done:
					default:
						close(done)
					}
					return
				}
			}
		}
	}()

	for {
		t, data, err := conn.ReadMessage()
		if err != nil {
			if isWSReadLimitError(err) {
				log.Printf("[ws.Handle] WARN reject type=frame userID=%d code=payload_too_large", uid)
				metrics.Add("ws_stroke_total{result=reject}", 1)
				metrics.Add("ws_stroke_reject_total{code=payload_too_large}", 1)
				globalHub.sendErrorToConn(conn, "payload_too_large", "message too large")
			} else if !isBenignNetErr(err) && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[ws.Handle] ERROR read userID=%d: %v", uid, err)
			}
			select {
			case <-done:
			default:
				close(done)
			}
			return
		}
		if t != websocket.TextMessage {
			continue
		}

		var m message
		if err := json.Unmarshal(data, &m); err != nil {
			log.Printf("[ws.Handle] WARN reject type=stroke userID=%d code=bad_json", uid)
			metrics.Add("ws_stroke_total{result=reject}", 1)
			metrics.Add("ws_stroke_reject_total{code=bad_json}", 1)
			globalHub.sendErrorToConn(conn, "bad_json", "invalid JSON message")
			continue
		}

		switch m.Type {
		case "stroke":
			handleStrokeMessage(conn, uid, m)
		case "delete":
			handleDeleteMessage(conn, uid, m)
		case "clear":
			handleClearMessage(conn, uid, m)
		default:
			log.Printf("[ws.Handle] DEBUG inbound type=%s userID=%d (ignored)", m.Type, uid)
		}
	}
}

func handleStrokeMessage(conn *websocket.Conn, uid int64, m message) {
	opID := m.OpID
	if opID == "" && m.Stroke != nil {
		opID = m.Stroke.OpID
	}
	if err := limits.ValidateOpID(opID); err != nil {
		nackStroke(conn, uid, opID, limits.ErrorCode(err), limits.SafeMessage(err), nil)
		return
	}
	baseRev, ok := requireBaseRev(conn, uid, opID, m.BaseRev, "stroke")
	if !ok {
		return
	}
	if m.Stroke == nil {
		nackStroke(conn, uid, opID, "invalid_stroke", "missing stroke object", nil)
		return
	}
	if !strokeLimiter.Allow(uid) {
		nackStroke(conn, uid, opID, "rate_limited", "too many strokes", nil)
		return
	}

	pts := make([]limits.FloatPoint, 0, len(m.Stroke.Points))
	for _, p := range m.Stroke.Points {
		pts = append(pts, limits.FloatPoint{X: p.X, Y: p.Y})
	}
	if err := limits.ValidateStrokePoints(pts); err != nil {
		nackStroke(conn, uid, opID, limits.ErrorCode(err), limits.SafeMessage(err), nil)
		return
	}
	if err := limits.ValidateStrokeMeta(m.Stroke.Width, m.Stroke.Color, m.Stroke.ClientID); err != nil {
		nackStroke(conn, uid, opID, limits.ErrorCode(err), limits.SafeMessage(err), nil)
		return
	}

	pointCount := len(m.Stroke.Points)
	log.Printf("[ws.Handle] DEBUG inbound type=stroke userID=%d opId=%s baseRev=%d points=%d", uid, opID, baseRev, pointCount)
	if m.Stroke.StartedAtUnixMs == 0 {
		m.Stroke.StartedAtUnixMs = time.Now().UnixMilli()
	}
	dbPts := make([]db.StrokePoint, 0, pointCount)
	for _, p := range m.Stroke.Points {
		dbPts = append(dbPts, db.StrokePoint{X: p.X, Y: p.Y})
	}
	result, err := globalHub.Store.ApplyStrokeCreate(uid, baseRev, opID, m.Stroke.Color, m.Stroke.Width, m.Stroke.StartedAtUnixMs, dbPts)
	if err != nil {
		if errors.Is(err, db.ErrStaleBoard) {
			rev := result.BoardRev
			nackStroke(conn, uid, opID, "stale_board", "board revision mismatch", &rev)
			return
		}
		if errors.Is(err, db.ErrOpCancelled) {
			rev := result.BoardRev
			nackStroke(conn, uid, opID, "op_cancelled", "create cancelled", &rev)
			return
		}
		log.Printf("[ws.Handle] ERROR save stroke userID=%d opId=%s: %v", uid, opID, err)
		nackStroke(conn, uid, opID, "internal_error", "failed to save stroke", nil)
		return
	}
	id := result.StrokeID
	boardRev := result.BoardRev
	m.Stroke.ID = id
	m.Stroke.OpID = opID
	m.OpID = opID
	m.BoardRev = &boardRev
	m.BaseRev = nil
	if result.Created {
		log.Printf("[ws.Handle] INFO stroke saved userID=%d id=%d opId=%s boardRev=%d", uid, id, opID, boardRev)
		metrics.Add("ws_stroke_total{result=ok}", 1)
	} else {
		log.Printf("[ws.Handle] INFO stroke idempotent hit userID=%d id=%d opId=%s boardRev=%d", uid, id, opID, boardRev)
		metrics.Add("ws_stroke_total{result=idempotent}", 1)
	}
	globalHub.sendAck(conn, opID, true, &boardRev, &id, nil, nil, "", "")
	globalHub.sendToUser(uid, m)
}

func handleDeleteMessage(conn *websocket.Conn, uid int64, m message) {
	opID := m.OpID
	if err := limits.ValidateOpID(opID); err != nil {
		ok := false
		globalHub.sendAck(conn, opID, ok, nil, nil, nil, nil, limits.ErrorCode(err), limits.SafeMessage(err))
		log.Printf("[ws.Handle] WARN reject type=delete userID=%d code=%s", uid, limits.ErrorCode(err))
		return
	}
	baseRev, ok := requireBaseRev(conn, uid, opID, m.BaseRev, "delete")
	if !ok {
		return
	}
	var strokeID int64
	if m.Delete != nil {
		strokeID = *m.Delete
	}
	deleteOpID := m.DeleteOpID
	if strokeID <= 0 && deleteOpID == "" {
		log.Printf("[ws.Handle] WARN reject type=delete userID=%d opId=%s code=invalid_stroke", uid, opID)
		globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, "invalid_stroke", "delete id or deleteOpId required")
		return
	}
	if deleteOpID != "" {
		if err := limits.ValidateOpID(deleteOpID); err != nil {
			globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, limits.ErrorCode(err), limits.SafeMessage(err))
			return
		}
	}
	log.Printf("[ws.Handle] DEBUG inbound type=delete userID=%d id=%d deleteOpId=%s opId=%s baseRev=%d",
		uid, strokeID, deleteOpID, opID, baseRev)
	result, err := globalHub.Store.ApplyStrokeDelete(uid, baseRev, opID, strokeID, deleteOpID)
	if err != nil {
		if errors.Is(err, db.ErrStaleBoard) {
			rev := result.BoardRev
			log.Printf("[ws.Handle] WARN reject type=delete userID=%d code=stale_board", uid)
			globalHub.sendAck(conn, opID, false, &rev, nil, nil, nil, "stale_board", "board revision mismatch")
			return
		}
		log.Printf("[ws.Handle] ERROR delete stroke userID=%d id=%d opId=%s: %v", uid, strokeID, opID, err)
		globalHub.sendAck(conn, opID, false, nil, nil, &strokeID, nil, "internal_error", "failed to delete stroke")
		return
	}
	boardRev := result.BoardRev
	log.Printf("[ws.Handle] INFO delete applied userID=%d id=%d deleteOpId=%s opId=%s boardRev=%d",
		uid, strokeID, deleteOpID, opID, boardRev)
	var delPtr *int64
	if strokeID > 0 {
		delPtr = &strokeID
	}
	globalHub.sendAck(conn, opID, true, &boardRev, nil, delPtr, nil, "", "")
	echo := message{Type: "delete", Delete: delPtr, DeleteOpID: deleteOpID, OpID: opID, BoardRev: &boardRev}
	globalHub.sendToUser(uid, echo)
}

func handleClearMessage(conn *websocket.Conn, uid int64, m message) {
	opID := m.OpID
	if err := limits.ValidateOpID(opID); err != nil {
		globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, limits.ErrorCode(err), limits.SafeMessage(err))
		log.Printf("[ws.Handle] WARN reject type=clear userID=%d code=%s", uid, limits.ErrorCode(err))
		return
	}
	baseRev, ok := requireBaseRev(conn, uid, opID, m.BaseRev, "clear")
	if !ok {
		return
	}
	log.Printf("[ws.Handle] DEBUG inbound type=clear userID=%d opId=%s baseRev=%d", uid, opID, baseRev)
	result, err := globalHub.Store.ApplyClear(uid, baseRev, opID)
	if err != nil {
		if errors.Is(err, db.ErrStaleBoard) {
			rev := result.BoardRev
			log.Printf("[ws.Handle] WARN reject type=clear userID=%d code=stale_board", uid)
			globalHub.sendAck(conn, opID, false, &rev, nil, nil, nil, "stale_board", "board revision mismatch")
			return
		}
		log.Printf("[ws.Handle] ERROR clear userID=%d opId=%s: %v", uid, opID, err)
		globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, "internal_error", "failed to clear board")
		return
	}
	boardRev := result.BoardRev
	cleared := true
	log.Printf("[ws.Handle] INFO clear applied userID=%d opId=%s boardRev=%d", uid, opID, boardRev)
	globalHub.sendAck(conn, opID, true, &boardRev, nil, nil, &cleared, "", "")
	echo := message{Type: "clear", OpID: opID, BoardRev: &boardRev, Clear: &cleared}
	globalHub.sendToUser(uid, echo)
}

func requireBaseRev(conn *websocket.Conn, uid int64, opID string, baseRev *int64, msgType string) (int64, bool) {
	if baseRev == nil {
		log.Printf("[ws.Handle] WARN reject type=%s userID=%d opId=%s code=invalid_stroke", msgType, uid, opID)
		globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, "invalid_stroke", "baseRev required")
		return 0, false
	}
	if *baseRev < 0 {
		log.Printf("[ws.Handle] WARN reject type=%s userID=%d opId=%s code=invalid_stroke", msgType, uid, opID)
		globalHub.sendAck(conn, opID, false, nil, nil, nil, nil, "invalid_stroke", "baseRev invalid")
		return 0, false
	}
	return *baseRev, true
}

func nackStroke(conn *websocket.Conn, uid int64, opID, code, message string, boardRev *int64) {
	log.Printf("[ws.Handle] WARN reject type=stroke userID=%d opId=%s code=%s", uid, opID, code)
	metrics.Add("ws_stroke_total{result=reject}", 1)
	metrics.Add("ws_stroke_reject_total{code="+code+"}", 1)
	globalHub.sendAck(conn, opID, false, boardRev, nil, nil, nil, code, message)
}

func rejectStroke(conn *websocket.Conn, uid int64, code, message string) {
	log.Printf("[ws.Handle] WARN reject type=stroke userID=%d code=%s", uid, code)
	metrics.Add("ws_stroke_total{result=reject}", 1)
	metrics.Add("ws_stroke_reject_total{code="+code+"}", 1)
	globalHub.sendErrorToConn(conn, code, message)
}

func isWSReadLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "read limit") || strings.Contains(msg, "message too big")
}

func isBenignNetErr(err error) bool {
	if err == nil {
		return false
	}
	var ne *net.OpError
	if errors.As(err, &ne) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}

// ValidateStrokeForTest runs shared stroke validation (unit tests).
func ValidateStrokeForTest(s *Stroke) error {
	if s == nil {
		return limits.ErrInvalidStroke
	}
	pts := make([]limits.FloatPoint, 0, len(s.Points))
	for _, p := range s.Points {
		pts = append(pts, limits.FloatPoint{X: p.X, Y: p.Y})
	}
	if err := limits.ValidateStrokePoints(pts); err != nil {
		return err
	}
	return limits.ValidateStrokeMeta(s.Width, s.Color, s.ClientID)
}
