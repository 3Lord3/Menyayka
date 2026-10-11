package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func newTestClient(hub *Hub, userID string) *Client {
	return &Client{
		hub:    hub,
		userID: userID,
		send:   make(chan []byte, sendBuffer),
		subs:   make(map[int64]time.Time),
	}
}

func drain(c *Client) []string {
	var messages []string
	for {
		select {
		case msg := <-c.send:
			messages = append(messages, string(msg))
		default:
			return messages
		}
	}
}

func TestHubRoutesChainRoomOnlyToSubscribers(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{SubTTL: time.Minute, MaxChainSubs: 5}}
	subscriber := newTestClient(hub, "u1")
	other := newTestClient(hub, "u2")
	hub.Register(subscriber)
	hub.Register(other)
	if !hub.Subscribe(subscriber, 5) {
		t.Fatal("Subscribe() = false, want true")
	}

	hub.Publish("chain:5", []byte("event"))

	if got := drain(subscriber); len(got) != 1 {
		t.Fatalf("subscriber received %v, want 1 message", got)
	}
	if got := drain(other); len(got) != 0 {
		t.Fatalf("non-subscriber received %v, want none", got)
	}
}

func TestHubEnforcesChainSubscriptionLimit(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{SubTTL: time.Minute, MaxChainSubs: 1}}
	client := newTestClient(hub, "u1")
	hub.Register(client)

	if !hub.Subscribe(client, 1) {
		t.Fatal("first Subscribe() = false, want true")
	}
	if hub.Subscribe(client, 2) {
		t.Fatal("second Subscribe() = true, want false (limit reached)")
	}
}

func TestHubPerUserConnectionLimit(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{MaxConnsPerUser: 1}}
	if !hub.TryAcquire("u1") {
		t.Fatal("first TryAcquire() = false, want true")
	}
	if hub.TryAcquire("u1") {
		t.Fatal("second TryAcquire() = true, want false")
	}
	if !hub.TryAcquire("u2") {
		t.Fatal("TryAcquire() for another user = false, want true")
	}
}

// A client that both subscribes to a chain and is one of its participants must not
// receive the same event twice.
func TestHubPublishRoomsDeduplicatesClients(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{SubTTL: time.Minute, MaxChainSubs: 5}}
	client := newTestClient(hub, "u1")
	hub.Register(client)
	if !hub.Subscribe(client, 5) {
		t.Fatal("Subscribe() = false, want true")
	}

	hub.PublishRooms([]string{"chain:5", "user:u1"}, []byte("event"))

	if got := drain(client); len(got) != 1 {
		t.Fatalf("client received %v, want exactly one message", got)
	}
}

// A double release must not drive the connection counter negative.
func TestHubReleaseIsIdempotent(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}}
	if !hub.TryAcquire("u1") {
		t.Fatal("TryAcquire() = false, want true")
	}
	hub.ReleaseAcquired("u1")
	hub.ReleaseAcquired("u1")

	if got := hub.ActiveConnections(); got != 0 {
		t.Fatalf("ActiveConnections() = %d, want 0", got)
	}
	if !hub.TryAcquire("u1") {
		t.Fatal("capacity was lost after a double release")
	}
}

// The global cap is enforced before any room bookkeeping, so a full hub rejects new
// users even when each of them is below the per-user limit.
func TestHubEnforcesGlobalConnectionLimit(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{MaxConnections: 1}}
	if !hub.TryAcquire("u1") {
		t.Fatal("first TryAcquire() = false, want true")
	}
	if hub.TryAcquire("u2") {
		t.Fatal("TryAcquire() at the global limit = true, want false")
	}
	hub.ReleaseAcquired("u1")
	if !hub.TryAcquire("u2") {
		t.Fatal("capacity was not freed after release")
	}
}

// A subscriber that cannot drain its buffer is closed with a "try again" code instead
// of blocking the fan-out goroutine.
func TestHubClosesSlowConsumer(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}}
	client := newTestClient(hub, "u1")
	dropped := make(chan string, 1)
	client.closeFunc = func(_ websocket.StatusCode, reason string) { dropped <- reason }
	hub.Register(client)

	for i := 0; i < sendBuffer; i++ {
		client.send <- []byte("filler")
	}
	hub.Publish("user:u1", []byte("overflow"))

	select {
	case reason := <-dropped:
		if reason != "slow consumer" {
			t.Fatalf("close reason = %q, want slow consumer", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("slow consumer was not closed")
	}
}

// Expired subscriptions are dropped and the client is told to resubscribe (reason=ttl),
// and the chain room no longer reaches it.
func TestHubPrunesExpiredSubscriptions(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{SubTTL: time.Minute, MaxChainSubs: 5}}
	client := newTestClient(hub, "u1")
	hub.Register(client)
	if !hub.Subscribe(client, 5) {
		t.Fatal("Subscribe() = false, want true")
	}
	client.subs[5] = time.Now().Add(-time.Second)

	hub.pruneExpired(client)
	hub.Publish("chain:5", []byte("event"))

	got := drain(client)
	if len(got) != 1 {
		t.Fatalf("received %v, want exactly the unsubscribed control", got)
	}
	var msg control
	if err := json.Unmarshal([]byte(got[0]), &msg); err != nil {
		t.Fatalf("control unmarshal: %v", err)
	}
	if msg.Action != "unsubscribed" || msg.Reason != "ttl" || msg.ChainID != 5 {
		t.Fatalf("control = %+v, want unsubscribed/ttl/5", msg)
	}
}

// Any accepted frame extends every subscription, so an active screen is not dropped.
func TestHubTouchKeepsSubscriptionsAlive(t *testing.T) {
	hub := &Hub{rooms: map[string]map[*Client]struct{}{}, clients: map[*Client]struct{}{}, perUser: map[string]int{}, cfg: HubConfig{SubTTL: time.Minute, MaxChainSubs: 5}}
	client := newTestClient(hub, "u1")
	hub.Register(client)
	if !hub.Subscribe(client, 5) {
		t.Fatal("Subscribe() = false, want true")
	}
	client.subs[5] = time.Now().Add(-time.Second)

	hub.Touch(client)
	hub.pruneExpired(client)

	if len(client.subs) != 1 {
		t.Fatalf("subscriptions = %d, want 1 after Touch", len(client.subs))
	}
}
