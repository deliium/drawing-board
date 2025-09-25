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
	CheckOrigin: func(r *http.Request) bool { return true },
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
	Type    string   `json:"type"`
	Stroke  *Stroke  `json:"stroke"`
	Delete  *int64   `json:"delete"` // stroke id to delete
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
	Store   *db.Store
	Auth    *auth.Service
}

func NewHub(store *db.Store, authSvc *auth.Service) *Hub { return &Hub{clients: make(map[*websocket.Conn]struct{}), Store: store, Auth: authSvc} }

func (h *Hub) add(c *websocket.Conn)    { h.mu.Lock(); h.clients[c] = struct{}{}; h.mu.Unlock() }
func (h *Hub) remove(c *websocket.Conn) { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock() }

func (h *Hub) broadcast(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil { return }
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			if !isBenignNetErr(err) {
				log.Printf("ws write error: %v", err)
			}
			c.Close()
			delete(h.clients, c)
		}
	}
}

var globalHub *Hub

func Init(store *db.Store, authSvc *auth.Service) { globalHub = NewHub(store, authSvc) }

func Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	log.Printf("ws connected: %s", r.RemoteAddr)
	globalHub.add(conn)
	defer func() {
		globalHub.remove(conn)
		conn.Close()
		log.Printf("ws disconnected: %s", r.RemoteAddr)
	}()

	conn.SetReadLimit(1 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	done := make(chan struct{})
	conn.SetCloseHandler(func(code int, text string) error {
		select { case <-done: default: close(done) }
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
						log.Printf("ws ping write error: %v", err)
					}
					_ = conn.Close()
					select { case <-done: default: close(done) }
					return
				}
			}
		}
	}()

	for {
		t, data, err := conn.ReadMessage()
		if err != nil {
			if !isBenignNetErr(err) && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("ws read: %v", err)
			}
			select { case <-done: default: close(done) }
			return
		}
		if t != websocket.TextMessage { continue }

		var m message
		if err := json.Unmarshal(data, &m); err != nil { log.Printf("ws bad json: %v", err); continue }

		switch m.Type {
		case "stroke":
			if m.Stroke == nil { continue }
			if m.Stroke.StartedAtUnixMs == 0 { m.Stroke.StartedAtUnixMs = time.Now().UnixMilli() }
			uid, ok := globalHub.Auth.UserIDFromRequest(r)
			if ok {
				pts := make([]db.StrokePoint, 0, len(m.Stroke.Points))
				for _, p := range m.Stroke.Points { pts = append(pts, db.StrokePoint{X:p.X, Y:p.Y}) }
				id, err := globalHub.Store.SaveStroke(uid, m.Stroke.Color, m.Stroke.Width, m.Stroke.StartedAtUnixMs, pts)
				if err != nil { log.Printf("save stroke: %v", err) } else { m.Stroke.ID = id }
			}
			globalHub.broadcast(m)
		case "delete":
			if m.Delete == nil { continue }
			uid, ok := globalHub.Auth.UserIDFromRequest(r)
			if ok { if err := globalHub.Store.DeleteStroke(uid, *m.Delete); err != nil { log.Printf("delete stroke: %v", err) } }
			globalHub.broadcast(m)
		}
	}
}

func isBenignNetErr(err error) bool {
	if err == nil { return false }
	var ne *net.OpError
	if errors.As(err, &ne) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}
