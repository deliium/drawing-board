package ws

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

const contractOrigin = "http://localhost:5173"

func dialAuthedWS(t *testing.T, serverURL string, authSvc *auth.Service, uid string) *websocket.Conn {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := authSvc.Sessions.Get(req, "sid")
	if err != nil {
		t.Fatalf("session get: %v", err)
	}
	sess.Values["user_id"] = uid
	if err := sess.Save(req, rec); err != nil {
		t.Fatalf("session save: %v", err)
	}
	header := http.Header{}
	header.Set("Origin", contractOrigin)
	var cookieParts []string
	for _, c := range rec.Result().Cookies() {
		cookieParts = append(cookieParts, c.Name+"="+c.Value)
	}
	header.Set("Cookie", strings.Join(cookieParts, "; "))

	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial: %v status=%d", err, resp.StatusCode)
		}
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestWSDial_CreateAckBoardRev(t *testing.T) {
	t.Logf("[ws.contract] case=create_ack_boardRev")
	path := filepath.Join(t.TempDir(), "ws-dial.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()

	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	authSvc := &auth.Service{Store: store, Sessions: sessionStore}
	uid, err := store.CreateUser("wsdial@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	Init(store, authSvc, []string{contractOrigin})
	SetStrokeLimiterForTest(limits.NewLimiter(1000, 1000))
	srv := httptest.NewServer(http.HandlerFunc(Handle))
	defer srv.Close()

	conn := dialAuthedWS(t, srv.URL, authSvc, uid)
	base := int64(0)
	create := message{
		Type:    "stroke",
		OpID:    "op-dial-1",
		BaseRev: &base,
		Stroke: &Stroke{
			Points:          []Point{{X: 1, Y: 1}, {X: 2, Y: 2}},
			Color:           "#111111",
			Width:           2,
			ClientID:        "dial",
			StartedAtUnixMs: 1,
		},
	}
	if err := conn.WriteJSON(create); err != nil {
		t.Fatalf("write create: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ack message
	gotAck := false
	for i := 0; i < 5; i++ {
		var m message
		if err := conn.ReadJSON(&m); err != nil {
			t.Fatalf("read: %v", err)
		}
		t.Logf("[ws.contract] frame type=%s ok=%v err=%s", m.Type, m.OK != nil && *m.OK, m.Error)
		if m.Type == "ack" && m.OpID == "op-dial-1" {
			ack = m
			gotAck = true
			break
		}
	}
	if !gotAck {
		t.Fatal("expected create ack")
	}
	if ack.OK == nil || !*ack.OK {
		t.Fatalf("ack not ok: %+v", ack)
	}
	if ack.BoardRev == nil || *ack.BoardRev != 1 {
		t.Fatalf("boardRev=%v want 1", ack.BoardRev)
	}
	if ack.StrokeID == nil || *ack.StrokeID == "" {
		t.Fatalf("strokeId missing: %+v", ack)
	}
}

func TestWSDial_StaleBoardNack(t *testing.T) {
	t.Logf("[ws.contract] case=stale_board")
	path := filepath.Join(t.TempDir(), "ws-stale.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()
	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	authSvc := &auth.Service{Store: store, Sessions: sessionStore}
	uid, err := store.CreateUser("wsstale@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if _, err := store.ApplyStrokeCreate(uid, 0, "op-seed", "#000000", 2, 1, []db.StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	Init(store, authSvc, []string{contractOrigin})
	SetStrokeLimiterForTest(limits.NewLimiter(1000, 1000))
	srv := httptest.NewServer(http.HandlerFunc(Handle))
	defer srv.Close()

	conn := dialAuthedWS(t, srv.URL, authSvc, uid)
	stale := int64(0)
	if err := conn.WriteJSON(message{
		Type:    "stroke",
		OpID:    "op-stale",
		BaseRev: &stale,
		Stroke: &Stroke{
			Points:   []Point{{X: 3, Y: 3}, {X: 4, Y: 4}},
			Color:    "#222222",
			Width:    2,
			ClientID: "c",
		},
	}); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < 5; i++ {
		var m message
		if err := conn.ReadJSON(&m); err != nil {
			t.Fatalf("read: %v", err)
		}
		if m.Type == "ack" && m.OpID == "op-stale" {
			if m.Error != "stale_board" {
				t.Fatalf("want stale_board got %+v", m)
			}
			t.Logf("[ws.contract] stale_board nack ok")
			return
		}
	}
	t.Fatal("expected stale_board ack")
}

func TestWSDial_SendToUserIsolation(t *testing.T) {
	t.Logf("[ws.contract] case=sendToUser_isolation")
	path := filepath.Join(t.TempDir(), "ws-iso.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.SQL.Close() }()
	sessionStore := sessions.NewCookieStore([]byte("test-cookie-key-32-bytes-minimum!!"))
	authSvc := &auth.Service{Store: store, Sessions: sessionStore}
	uidA, err := store.CreateUser("wsa@example.com", "hash")
	if err != nil {
		t.Fatalf("userA: %v", err)
	}
	uidB, err := store.CreateUser("wsb@example.com", "hash")
	if err != nil {
		t.Fatalf("userB: %v", err)
	}

	Init(store, authSvc, []string{contractOrigin})
	SetStrokeLimiterForTest(limits.NewLimiter(1000, 1000))
	srv := httptest.NewServer(http.HandlerFunc(Handle))
	defer srv.Close()

	connA := dialAuthedWS(t, srv.URL, authSvc, uidA)
	connB := dialAuthedWS(t, srv.URL, authSvc, uidB)

	base := int64(0)
	if err := connA.WriteJSON(message{
		Type:    "stroke",
		OpID:    "op-iso-a",
		BaseRev: &base,
		Stroke: &Stroke{
			Points:   []Point{{X: 1, Y: 1}, {X: 2, Y: 2}},
			Color:    "#111111",
			Width:    2,
			ClientID: "a",
		},
	}); err != nil {
		t.Fatalf("write A: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	_ = connA.SetReadDeadline(deadline)
	_ = connB.SetReadDeadline(time.Now().Add(400 * time.Millisecond))

	sawStrokeOnA := false
	for {
		var m message
		if err := connA.ReadJSON(&m); err != nil {
			break
		}
		if m.Type == "stroke" && m.Stroke != nil {
			sawStrokeOnA = true
			break
		}
		if m.Type == "ack" && m.OpID == "op-iso-a" && m.OK != nil && *m.OK {
			// continue for echo
			continue
		}
	}
	if !sawStrokeOnA {
		// ack may be enough; isolation check is that B gets nothing
		t.Logf("[ws.contract] no stroke echo on A (ack-only path ok)")
	}

	var leaked message
	if err := connB.ReadJSON(&leaked); err == nil {
		t.Fatalf("user B must not receive A's frames, got type=%s", leaked.Type)
	}
	t.Logf("[ws.contract] isolation ok (B read timed out)")
}

func TestHub_RaceStressSendToUser(t *testing.T) {
	t.Logf("[ws.race] concurrent sendToUser + add/remove")
	hub := NewHub(&db.Store{}, &auth.Service{})
	var mu sync.Mutex
	writes := 0
	hub.writeFn = func(c *websocket.Conn, data []byte) error {
		mu.Lock()
		writes++
		mu.Unlock()
		_ = data
		return nil
	}

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			conn := &websocket.Conn{}
			hub.add(conn, fmt.Sprintf("user-%d", id%4+1))
			for j := 0; j < 50; j++ {
				strokeID := fmt.Sprintf("stroke-%d", j)
				hub.sendToUser(fmt.Sprintf("user-%d", j%4+1), message{Type: "stroke", Stroke: &Stroke{ID: strokeID, Points: []Point{{X: 1, Y: 1}}}})
			}
			hub.remove(conn)
		}(i)
	}
	wg.Wait()
	t.Logf("[ws.race] writes=%d clients=%d", writes, len(hub.clients))
	if len(hub.clients) != 0 {
		t.Fatalf("expected empty hub, got %d", len(hub.clients))
	}
}
