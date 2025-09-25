package auth

import (
	"testing"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/sessions"
)

func TestNewService(t *testing.T) {
	store := &db.Store{}
	sessionStore := sessions.NewCookieStore([]byte("test-secret"))
	
	service := NewService(store, sessionStore)
	if service == nil {
		t.Fatal("Service should not be nil")
	}
	
	if service.Store != store {
		t.Fatal("Store should be set correctly")
	}
	
	if service.Sessions != sessionStore {
		t.Fatal("Sessions should be set correctly")
	}
}

func TestService_Structure(t *testing.T) {
	store := &db.Store{}
	sessionStore := sessions.NewCookieStore([]byte("test-secret"))
	service := NewService(store, sessionStore)
	
	// Test that service has expected fields
	if service.Store == nil {
		t.Fatal("Store should not be nil")
	}
	
	if service.Sessions == nil {
		t.Fatal("Sessions should not be nil")
	}
}