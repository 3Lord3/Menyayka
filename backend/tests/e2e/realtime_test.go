//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func wsEndpoint() string {
	return "ws" + strings.TrimPrefix(baseURL(), "http") + "/api/v1/ws"
}

// TestE2E_RealtimeTicketAndItemEvent walks the realtime slice over the wire:
// ticket -> upgrade -> ping/pong -> a REST mutation delivered as a push event.
func TestE2E_RealtimeTicketAndItemEvent(t *testing.T) {
	waitHealthy(t)

	email := fmt.Sprintf("e2e-ws-%d@example.com", time.Now().UnixNano())
	status, reg := doJSON(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"fullName": "WS User",
		"email":    email,
		"password": "password123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("register: want 201, got %d body=%v", status, reg)
	}
	token, _ := reg["token"].(string)
	if token == "" {
		t.Fatalf("register: empty token: %v", reg)
	}

	status, item := doJSON(t, http.MethodPost, "/api/v1/items", map[string]any{
		"title":       "WS Bike",
		"description": "realtime test",
		"category":    "sport",
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("create item: want 201, got %d body=%v", status, item)
	}
	itemID, ok := item["id"].(float64)
	if !ok || itemID < 1 {
		t.Fatalf("create item: bad id: %v", item)
	}

	// The ticket endpoint must be authenticated.
	if status, _ := doJSON(t, http.MethodGet, "/api/v1/ws/ticket", nil, ""); status != http.StatusUnauthorized {
		t.Fatalf("ticket without auth: want 401, got %d", status)
	}

	status, ticketBody := doJSON(t, http.MethodGet, "/api/v1/ws/ticket", nil, token)
	if status != http.StatusOK {
		t.Fatalf("ticket: want 200, got %d body=%v", status, ticketBody)
	}
	ticket, _ := ticketBody["ticket"].(string)
	if ticket == "" {
		t.Fatalf("ticket: empty: %v", ticketBody)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsEndpoint(), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.CloseNow()

	// the one-time ticket is burned by the upgrade
	if _, _, err := websocket.Dial(ctx, wsEndpoint(), &websocket.DialOptions{
		Subprotocols: []string{"bearer", ticket},
	}); err == nil {
		t.Fatal("a reused ticket must not open a second connection")
	}

	// ping/pong proves the message loop is live (the connection is registered)
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"action":"ping"}`)); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if !strings.Contains(string(data), "pong") {
		t.Fatalf("ping reply = %s, want pong", data)
	}

	// a REST mutation must reach the connected client as a push event
	status, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/items/%d", int64(itemID)), nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("archive item: want 204, got %d", status)
	}

	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	var event struct {
		Type    string `json:"type"`
		Payload struct {
			ItemID int64  `json:"itemId"`
			Status string `json:"status"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode event %s: %v", data, err)
	}
	if event.Type != "item.status_changed" {
		t.Fatalf("event type = %q, want item.status_changed (raw=%s)", event.Type, data)
	}
	if event.Payload.ItemID != int64(itemID) {
		t.Fatalf("event itemId = %d, want %d", event.Payload.ItemID, int64(itemID))
	}
	if event.Payload.Status != "ARCHIVED" {
		t.Fatalf("event status = %q, want ARCHIVED", event.Payload.Status)
	}
}
