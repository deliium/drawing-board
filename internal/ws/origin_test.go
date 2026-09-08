package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOriginAllowlist(t *testing.T) {
	SetAllowedOriginsForTest([]string{"http://localhost:5173", "http://127.0.0.1:5173"})
	t.Cleanup(func() { SetAllowedOriginsForTest(nil) })

	t.Run("allowlisted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		if !CheckOriginForTest(req) {
			t.Fatal("expected allow")
		}
	})

	t.Run("evil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		if CheckOriginForTest(req) {
			t.Fatal("expected reject")
		}
	})

	t.Run("missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		if CheckOriginForTest(req) {
			t.Fatal("expected reject missing Origin")
		}
	})
}
