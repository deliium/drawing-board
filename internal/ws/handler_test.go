package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/websocket"
)

func TestHub_Add(t *testing.T) {
	// Create a mock store and auth service
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	conn := &websocket.Conn{}
	
	hub.add(conn)
	
	if len(hub.clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(hub.clients))
	}
	
	if _, exists := hub.clients[conn]; !exists {
		t.Fatal("Client should be registered")
	}
}

func TestHub_Remove(t *testing.T) {
	// Create a mock store and auth service
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	conn := &websocket.Conn{}
	
	// Add first
	hub.add(conn)
	if len(hub.clients) != 1 {
		t.Fatalf("Expected 1 client after add, got %d", len(hub.clients))
	}
	
	// Remove
	hub.remove(conn)
	if len(hub.clients) != 0 {
		t.Fatalf("Expected 0 clients after remove, got %d", len(hub.clients))
	}
}

func TestHub_Broadcast(t *testing.T) {
	// Create a mock store and auth service
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	
	// Create a test message
	msg := message{
		Type: "stroke",
		Stroke: &Stroke{
			ID:     1,
			Points: []Point{{X: 10, Y: 20}},
			Color:  "#000000",
			Width:  2,
		},
	}
	
	// Broadcast should not panic with no clients
	hub.broadcast(msg)
	
	// This is a basic test - in a real scenario, we'd need to mock WebSocket connections
	// to test actual message sending
}

func TestHub_ConcurrentOperations(t *testing.T) {
	// Create a mock store and auth service
	store := &db.Store{}
	authSvc := &auth.Service{}
	hub := NewHub(store, authSvc)
	
	// Test concurrent register/unregister
	done := make(chan bool)
	
	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			conn := &websocket.Conn{}
			hub.add(conn)
			time.Sleep(1 * time.Millisecond)
			hub.remove(conn)
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have no clients left
	if len(hub.clients) != 0 {
		t.Fatalf("Expected 0 clients after concurrent operations, got %d", len(hub.clients))
	}
}

func TestMessage_JSON(t *testing.T) {
	// Test stroke message
	strokeMsg := message{
		Type: "stroke",
		Stroke: &Stroke{
			ID:     1,
			Points: []Point{{X: 10, Y: 20}, {X: 30, Y: 40}},
			Color:  "#000000",
			Width:  2,
		},
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(strokeMsg)
	if err != nil {
		t.Fatalf("Failed to marshal stroke message: %v", err)
	}
	
	// Unmarshal back
	var unmarshaled message
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stroke message: %v", err)
	}
	
	// Check values
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
	
	// Marshal to JSON
	jsonData, err := json.Marshal(stroke)
	if err != nil {
		t.Fatalf("Failed to marshal stroke: %v", err)
	}
	
	// Unmarshal back
	var unmarshaled Stroke
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stroke: %v", err)
	}
	
	// Check values
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
	
	// Marshal to JSON
	jsonData, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("Failed to marshal point: %v", err)
	}
	
	// Unmarshal back
	var unmarshaled Point
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal point: %v", err)
	}
	
	// Check values
	if unmarshaled.X != 10.5 {
		t.Fatalf("Expected X 10.5, got %f", unmarshaled.X)
	}
	
	if unmarshaled.Y != 20.5 {
		t.Fatalf("Expected Y 20.5, got %f", unmarshaled.Y)
	}
}