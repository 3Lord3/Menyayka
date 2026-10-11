package realtime_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/core/config"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/codestore"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/token"
)

// fakeAccess allows a fixed set of chain IDs.
type fakeAccess struct {
	allowed map[int64]bool
}

func (f fakeAccess) IsChainParticipant(_ context.Context, _ string, chainID int64) (bool, error) {
	return f.allowed[chainID], nil
}

func newTestHandlerWithConfig(t *testing.T, cfg *config.Config) (*httptest.Server, *codestore.Store, uuid.UUID, *realtime.Hub) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokens := token.NewService("test-secret", time.Hour)
	store := codestore.New()
	hub := realtime.NewHub(context.Background(), realtime.HubConfig{
		Heartbeat:       cfg.WSHeartbeat,
		WriteTimeout:    cfg.WSWriteTimeout,
		MaxChainSubs:    cfg.WSMaxChainSubs,
		SubTTL:          cfg.WSSubTTL,
		MaxConnections:  cfg.WSMaxConnections,
		MsgRate:         cfg.WSMsgRate,
		MsgRateWindow:   cfg.WSMsgRateWindow,
		MaxConnsPerUser: cfg.WSMaxConnsPerUser,
	}, logrus.New())
	handler := realtime.NewHandler(hub, tokens, store, fakeAccess{allowed: map[int64]bool{1: true, 7: true}}, cfg, logrus.New())

	userID := uuid.New()
	engine := gin.New()
	engine.GET("/api/v1/ws/ticket", func(c *gin.Context) {
		c.Set(api.UserIDContextKey, userID)
		handler.Ticket(c)
	})
	engine.GET("/api/v1/ws", handler.Connect)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	t.Cleanup(hub.Close)
	return srv, store, userID, hub
}

func testWSConfig(allowedOrigins []string) *config.Config {
	return &config.Config{
		WSTicketTTL:       30 * time.Second,
		WSAllowedOrigins:  allowedOrigins,
		WSHeartbeat:       time.Minute,
		WSWriteTimeout:    5 * time.Second,
		WSMaxChainSubs:    5,
		WSSubTTL:          time.Minute,
		WSMsgRate:         20,
		WSMsgRateWindow:   10 * time.Second,
		WSMaxConnections:  0,
		WSMaxConnsPerUser: 5,
	}
}

func newTestHandler(t *testing.T, allowedOrigins []string) (*httptest.Server, *codestore.Store, uuid.UUID) {
	t.Helper()
	srv, store, userID, _ := newTestHandlerWithConfig(t, testWSConfig(allowedOrigins))
	return srv, store, userID
}

func fetchTicket(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/v1/ws/ticket")
	if err != nil {
		t.Fatalf("GET ticket: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET ticket status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}
	if body.Ticket == "" {
		t.Fatal("ticket is empty")
	}
	return body.Ticket
}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
}

func TestHandlerAcceptsUpgradeWithOneTimeTicket(t *testing.T) {
	srv, _, userID := newTestHandler(t, []string{"http://localhost:5173"})
	ticket := fetchTicket(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	})
	if err != nil {
		t.Fatalf("Dial with valid ticket failed: %v", err)
	}
	defer conn.CloseNow()
	if userID == uuid.Nil {
		t.Fatal("unreachable")
	}

	// subscribe ack proves the client is registered
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"action":"subscribe","chainId":1}`)); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if !strings.Contains(string(data), "subscribed") {
		t.Fatalf("ack = %s, want subscribed", data)
	}
}

func TestHandlerRejectsTicketReuse(t *testing.T) {
	srv, _, _ := newTestHandler(t, []string{"http://localhost:5173"})
	ticket := fetchTicket(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	})
	if err != nil {
		t.Fatalf("first Dial failed: %v", err)
	}
	conn.CloseNow()

	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	}); err == nil {
		t.Fatal("reused one-time ticket must be rejected")
	}
}

func TestHandlerRejectsDisallowedOrigin(t *testing.T) {
	srv, _, _ := newTestHandler(t, nil) // fail-closed: empty whitelist
	ticket := fetchTicket(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
		HTTPHeader:   http.Header{"Origin": []string{"http://evil.example"}},
	}); err == nil {
		t.Fatal("handshake from a disallowed origin must fail")
	}
}

func TestHandlerRejectsUnknownPurposeTicket(t *testing.T) {
	// a session token must not be usable as a ws ticket
	tokens := token.NewService("test-secret", time.Hour)
	session, err := tokens.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	srv, _, _ := newTestHandler(t, []string{"http://localhost:5173"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", session},
	}); err == nil {
		t.Fatal("session token must not authenticate a ws upgrade")
	}
}

// The fan-out must not expose chains the REST read path hides from non-participants.
func TestHandlerRejectsSubscriptionToForeignChain(t *testing.T) {
	srv, _, _ := newTestHandler(t, []string{"http://localhost:5173"})
	ticket := fetchTicket(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	})
	if err != nil {
		t.Fatalf("Dial with valid ticket failed: %v", err)
	}
	defer conn.CloseNow()

	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"action":"subscribe","chainId":99}`)); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if !strings.Contains(string(data), "forbidden") {
		t.Fatalf("ack = %s, want a forbidden error", data)
	}

	// an allowed chain still subscribes afterwards
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"action":"subscribe","chainId":7}`)); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if !strings.Contains(string(data), "subscribed") {
		t.Fatalf("ack = %s, want subscribed", data)
	}
}

// A full hub answers 503 before the upgrade consumes a slot.
func TestHandlerRejectsWhenGlobalConnectionLimitReached(t *testing.T) {
	cfg := testWSConfig([]string{"http://localhost:5173"})
	cfg.WSMaxConnections = 1
	srv, _, _, _ := newTestHandlerWithConfig(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	first, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", fetchTicket(t, srv)},
	})
	if err != nil {
		t.Fatalf("first Dial failed: %v", err)
	}
	defer first.CloseNow()

	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", fetchTicket(t, srv)},
	}); err == nil {
		t.Fatal("second Dial at the global connection limit must fail")
	}
}

// A rejected connection still burns the ticket, so a retry needs a fresh one.
func TestHandlerBurnsTicketWhenConnectionRejected(t *testing.T) {
	cfg := testWSConfig([]string{"http://localhost:5173"})
	cfg.WSMaxConnections = 1
	srv, _, _, _ := newTestHandlerWithConfig(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	first, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", fetchTicket(t, srv)},
	})
	if err != nil {
		t.Fatalf("first Dial failed: %v", err)
	}
	defer first.CloseNow()

	ticket := fetchTicket(t, srv)
	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	}); err == nil {
		t.Fatal("Dial at the limit must fail")
	}
	// even after the slot is released the ticket must not be reusable
	first.CloseNow()
	if _, _, err := websocket.Dial(ctx, wsURL(srv), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	}); err == nil {
		t.Fatal("a rejected ticket must not be reusable after capacity frees up")
	}
}
