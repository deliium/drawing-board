package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/websocket"
)

func TestHub_Add(t *testing.T) {
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	conn := &websocket.Conn{}

	hub.add(conn, 42)

	if len(hub.clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(hub.clients))
	}

	uid, exists := hub.clients[conn]
	if !exists {
		t.Fatal("Client should be registered")
	}
	if uid != 42 {
		t.Fatalf("Expected userID 42, got %d", uid)
	}
}

func TestHub_Remove(t *testing.T) {
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	conn := &websocket.Conn{}

	hub.add(conn, 7)
	if len(hub.clients) != 1 {
		t.Fatalf("Expected 1 client after add, got %d", len(hub.clients))
	}

	hub.remove(conn)
	if len(hub.clients) != 0 {
		t.Fatalf("Expected 0 clients after remove, got %d", len(hub.clients))
	}
}

func TestHub_SendToUser_NoClients(t *testing.T) {
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)

	msg := message{
		Type: "stroke",
		Stroke: &Stroke{
			ID:     1,
			Points: []Point{{X: 10, Y: 20}},
			Color:  "#000000",
			Width:  2,
		},
	}

	// Should not panic with no clients
	hub.sendToUser(1, msg)
}

func TestHub_SendToUser_Isolation(t *testing.T) {
	t.Logf("setup: two users, user A has two connections (multi-tab), user B has one")
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)

	connA1 := &websocket.Conn{}
	connA2 := &websocket.Conn{}
	connB := &websocket.Conn{}
	hub.add(connA1, 1)
	hub.add(connA2, 1)
	hub.add(connB, 2)

	var mu sync.Mutex
	delivered := map[*websocket.Conn]int{}
	hub.writeFn = func(c *websocket.Conn, data []byte) error {
		mu.Lock()
		delivered[c]++
		mu.Unlock()
		return nil
	}

	strokeMsg := message{
		Type: "stroke",
		Stroke: &Stroke{
			ID:              10,
			Points:          []Point{{X: 1, Y: 2}},
			Color:           "#111111",
			Width:           3,
			ClientID:        "a",
			StartedAtUnixMs: 100,
		},
	}
	hub.sendToUser(1, strokeMsg)

	mu.Lock()
	a1 := delivered[connA1]
	a2 := delivered[connA2]
	b := delivered[connB]
	mu.Unlock()

	if a1 != 1 || a2 != 1 {
		t.Fatalf("expected both user A connections to receive stroke (got A1=%d A2=%d); isolation requires multi-tab echo for owning user", a1, a2)
	}
	if b != 0 {
		t.Fatalf("expected user B to receive 0 stroke messages, got %d; cross-user isolation violated", b)
	}

	// Reset and verify delete isolation
	mu.Lock()
	delivered = map[*websocket.Conn]int{}
	mu.Unlock()

	delID := int64(10)
	deleteMsg := message{Type: "delete", Delete: &delID}
	hub.sendToUser(1, deleteMsg)

	mu.Lock()
	a1 = delivered[connA1]
	a2 = delivered[connA2]
	b = delivered[connB]
	mu.Unlock()

	if a1 != 1 || a2 != 1 {
		t.Fatalf("expected both user A connections to receive delete (got A1=%d A2=%d)", a1, a2)
	}
	if b != 0 {
		t.Fatalf("expected user B to receive 0 delete messages, got %d; delete isolation violated", b)
	}
}

func TestHub_SendToUser_OnlyTargetUser(t *testing.T) {
	t.Logf("setup: send to user B must not reach user A")
	hub := NewHub(&db.Store{}, &auth.Service{})
	connA := &websocket.Conn{}
	connB := &websocket.Conn{}
	hub.add(connA, 1)
	hub.add(connB, 2)

	var mu sync.Mutex
	delivered := map[*websocket.Conn]int{}
	hub.writeFn = func(c *websocket.Conn, data []byte) error {
		mu.Lock()
		delivered[c]++
		mu.Unlock()
		return nil
	}

	delID := int64(99)
	hub.sendToUser(2, message{Type: "delete", Delete: &delID})

	mu.Lock()
	defer mu.Unlock()
	if delivered[connB] != 1 {
		t.Fatalf("expected user B recipient count 1, got %d", delivered[connB])
	}
	if delivered[connA] != 0 {
		t.Fatalf("expected user A recipient count 0, got %d; must not receive other user's deletes", delivered[connA])
	}
}

func TestHub_ConcurrentOperations(t *testing.T) {
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(userID int64) {
			conn := &websocket.Conn{}
			hub.add(conn, userID)
			time.Sleep(1 * time.Millisecond)
			hub.remove(conn)
			done <- true
		}(int64(i + 1))
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if len(hub.clients) != 0 {
		t.Fatalf("Expected 0 clients after concurrent operations, got %d", len(hub.clients))
	}
}

func TestMessage_JSON(t *testing.T) {
	strokeMsg := message{
		Type: "stroke",
		Stroke: &Stroke{
			ID:     1,
			Points: []Point{{X: 10, Y: 20}, {X: 30, Y: 40}},
			Color:  "#000000",
			Width:  2,
		},
	}

	jsonData, err := json.Marshal(strokeMsg)
	if err != nil {
		t.Fatalf("Failed to marshal stroke message: %v", err)
	}

	var unmarshaled message
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stroke message: %v", err)
	}

	if unmarshaled.Type != "stroke" {
		t.Fatalf("Expected type 'stroke', got '%s'", unmarshaled.Type)
	}

	if unmarshaled.Stroke.ID != 1 {
		t.Fatalf("Expected stroke ID 1, got %d", unmarshaled.Stroke.ID)
	}

	if len(unmarshaled.Stroke.Points) != 2 {
		t.Fatalf("Expected 2 points, got %d", len(unmarshaled.Stroke.Points))
	}
}

func TestStroke_JSON(t *testing.T) {
	stroke := Stroke{
		ID:     1,
		Points: []Point{{X: 10, Y: 20}, {X: 30, Y: 40}},
		Color:  "#000000",
		Width:  2,
	}

	jsonData, err := json.Marshal(stroke)
	if err != nil {
		t.Fatalf("Failed to marshal stroke: %v", err)
	}

	var unmarshaled Stroke
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stroke: %v", err)
	}

	if unmarshaled.ID != 1 {
		t.Fatalf("Expected ID 1, got %d", unmarshaled.ID)
	}

	if len(unmarshaled.Points) != 2 {
		t.Fatalf("Expected 2 points, got %d", len(unmarshaled.Points))
	}

	if unmarshaled.Points[0].X != 10 {
		t.Fatalf("Expected first point X 10, got %f", unmarshaled.Points[0].X)
	}

	if unmarshaled.Points[0].Y != 20 {
		t.Fatalf("Expected first point Y 20, got %f", unmarshaled.Points[0].Y)
	}

	if unmarshaled.Color != "#000000" {
		t.Fatalf("Expected color '#000000', got '%s'", unmarshaled.Color)
	}

	if unmarshaled.Width != 2 {
		t.Fatalf("Expected width 2, got %d", unmarshaled.Width)
	}
}

func TestPoint_JSON(t *testing.T) {
	point := Point{X: 10.5, Y: 20.5}

	jsonData, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("Failed to marshal point: %v", err)
	}

	var unmarshaled Point
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal point: %v", err)
	}

	if unmarshaled.X != 10.5 {
		t.Fatalf("Expected X 10.5, got %f", unmarshaled.X)
	}

	if unmarshaled.Y != 20.5 {
		t.Fatalf("Expected Y 20.5, got %f", unmarshaled.Y)
	}
}

func TestSendAck_ToConnOnly(t *testing.T) {
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)

	connA := &websocket.Conn{}
	connB := &websocket.Conn{}
	hub.add(connA, 1)
	hub.add(connB, 1)

	var mu sync.Mutex
	delivered := map[*websocket.Conn]int{}
	hub.writeFn = func(c *websocket.Conn, data []byte) error {
		mu.Lock()
		delivered[c]++
		mu.Unlock()
		var m message
		_ = json.Unmarshal(data, &m)
		if m.Type != "ack" || m.OpID != "op-1" || m.OK == nil || !*m.OK {
			t.Errorf("unexpected ack payload: %+v", m)
		}
		return nil
	}

	id := int64(42)
	hub.sendAck(connA, "op-1", true, &id, nil, "", "")

	mu.Lock()
	defer mu.Unlock()
	if delivered[connA] != 1 {
		t.Fatalf("expected ack only on connA, got %d", delivered[connA])
	}
	if delivered[connB] != 0 {
		t.Fatalf("ack must not fan out to other tabs via sendAck, got %d", delivered[connB])
	}
}
