package ws

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
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
}

type message struct {
	Type   string  `json:"type"`
	Stroke *Stroke `json:"stroke"`
	Delete *int64  `json:"delete"` // stroke id to delete
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

var globalHub *Hub

func Init(store *db.Store, authSvc *auth.Service) { globalHub = NewHub(store, authSvc) }

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

	conn.SetReadLimit(1 << 20)
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
			if !isBenignNetErr(err) && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
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
			log.Printf("[ws.Handle] WARN bad json userID=%d: %v", uid, err)
			continue
		}

		switch m.Type {
		case "stroke":
			if m.Stroke == nil {
				continue
			}
			pointCount := len(m.Stroke.Points)
			log.Printf("[ws.Handle] DEBUG inbound type=stroke userID=%d points=%d", uid, pointCount)
			if m.Stroke.StartedAtUnixMs == 0 {
				m.Stroke.StartedAtUnixMs = time.Now().UnixMilli()
			}
			pts := make([]db.StrokePoint, 0, pointCount)
			for _, p := range m.Stroke.Points {
				pts = append(pts, db.StrokePoint{X: p.X, Y: p.Y})
			}
			id, err := globalHub.Store.SaveStroke(uid, m.Stroke.Color, m.Stroke.Width, m.Stroke.StartedAtUnixMs, pts)
			if err != nil {
				log.Printf("[ws.Handle] ERROR save stroke userID=%d: %v", uid, err)
			} else {
				m.Stroke.ID = id
				log.Printf("[ws.Handle] INFO stroke saved userID=%d id=%d", uid, id)
			}
			globalHub.sendToUser(uid, m)
		case "delete":
			if m.Delete == nil {
				continue
			}
			log.Printf("[ws.Handle] DEBUG inbound type=delete userID=%d id=%d", uid, *m.Delete)
			if err := globalHub.Store.DeleteStroke(uid, *m.Delete); err != nil {
				log.Printf("[ws.Handle] ERROR delete stroke userID=%d id=%d: %v", uid, *m.Delete, err)
			} else {
				log.Printf("[ws.Handle] INFO delete applied userID=%d id=%d", uid, *m.Delete)
			}
			globalHub.sendToUser(uid, m)
		default:
			log.Printf("[ws.Handle] DEBUG inbound type=%s userID=%d (ignored)", m.Type, uid)
		}
	}
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
