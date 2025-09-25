package httpapi

import (
	"testing"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
)

func TestNewAPI(t *testing.T) {
	authService := &auth.Service{}
	store := &db.Store{}
	
	api := &API{
		Auth:  authService,
		Store: store,
	}
	
	if api == nil {
		t.Fatal("API should not be nil")
	}
	
	if api.Auth != authService {
		t.Fatal("Auth should be set correctly")
	}
	
	if api.Store != store {
		t.Fatal("Store should be set correctly")
	}
}