package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HubConfig holds the hub's runtime limits and timings.
type HubConfig struct {
	Heartbeat       time.Duration
	WriteTimeout    time.Duration
	MaxChainSubs    int
	SubTTL          time.Duration
	MaxConnections  int
	MaxConnsPerUser int
	MsgRate         int
	MsgRateWindow   time.Duration
}

// Hub is an in-memory room registry, safe for concurrent use. Single-replica only.
type Hub struct {
	ctx    context.Context
	cfg    HubConfig
	logger *logrus.Logger

	mu      sync.RWMutex
	rooms   map[string]map[*Client]struct{}
	clients map[*Client]struct{}
	conns   int
	perUser map[string]int

	closeOnce sync.Once
}

// NewHub creates a hub bound to ctx (cancel closes every connection). Non-positive
// durations/limits fall back to safe defaults.
func NewHub(ctx context.Context, cfg HubConfig, logger *logrus.Logger) *Hub {
	if cfg.Heartbeat <= 0 {
		cfg.Heartbeat = 30 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.SubTTL <= 0 {
		cfg.SubTTL = 30 * time.Minute
	}
	if cfg.MsgRateWindow <= 0 {
		cfg.MsgRateWindow = 10 * time.Second
	}
	if cfg.MsgRate < 0 {
		cfg.MsgRate = 0
	}
	h := &Hub{
		ctx:     ctx,
		cfg:     cfg,
		logger:  logger,
		rooms:   make(map[string]map[*Client]struct{}),
		clients: make(map[*Client]struct{}),
		perUser: make(map[string]int),
	}
	go func() {
		<-ctx.Done()
		h.Close()
	}()
	return h
}

// TryAcquire reserves a connection slot before the upgrade; false when a limit is hit.
func (h *Hub) TryAcquire(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg.MaxConnections > 0 && h.conns >= h.cfg.MaxConnections {
		return false
	}
	if h.cfg.MaxConnsPerUser > 0 && h.perUser[userID] >= h.cfg.MaxConnsPerUser {
		return false
	}
	h.conns++
	h.perUser[userID]++
	return true
}

// ReleaseAcquired undoes TryAcquire when the upgrade did not complete.
func (h *Hub) ReleaseAcquired(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.releaseUserLocked(userID)
}

// releaseUserLocked frees a reserved slot; the guard keeps a double release from
// driving the counter negative.
func (h *Hub) releaseUserLocked(userID string) {
	if h.perUser[userID] <= 0 {
		return
	}
	h.conns--
	if n := h.perUser[userID] - 1; n > 0 {
		h.perUser[userID] = n
	} else {
		delete(h.perUser, userID)
	}
}

func (h *Hub) addRoomLocked(room string, c *Client) {
	set := h.rooms[room]
	if set == nil {
		set = make(map[*Client]struct{})
		h.rooms[room] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) removeRoomLocked(room string, c *Client) {
	set := h.rooms[room]
	if set == nil {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.rooms, room)
	}
}

// Register adds a client to its user room.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	h.addRoomLocked(userRoom(c.userID), c)
}

// Unregister removes a client from every room and releases its connection slot.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	h.removeRoomLocked(userRoom(c.userID), c)
	for chainID := range c.subs {
		h.removeRoomLocked(chainRoom(chainID), c)
		delete(c.subs, chainID)
	}
	h.releaseUserLocked(c.userID)
}

// Subscribe adds a chain subscription if the per-connection limit is not exceeded.
func (h *Hub) Subscribe(c *Client, chainID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := c.subs[chainID]; !exists && h.cfg.MaxChainSubs > 0 && len(c.subs) >= h.cfg.MaxChainSubs {
		return false
	}
	c.subs[chainID] = time.Now().Add(h.cfg.SubTTL)
	h.addRoomLocked(chainRoom(chainID), c)
	return true
}

// Unsubscribe removes a chain subscription.
func (h *Hub) Unsubscribe(c *Client, chainID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := c.subs[chainID]; !exists {
		return
	}
	delete(c.subs, chainID)
	h.removeRoomLocked(chainRoom(chainID), c)
}

// Touch extends the TTL of every subscription of a client.
func (h *Hub) Touch(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	expiresAt := time.Now().Add(h.cfg.SubTTL)
	for chainID := range c.subs {
		c.subs[chainID] = expiresAt
	}
}

// pruneExpired drops elapsed subscriptions and notifies the client to resubscribe.
func (h *Hub) pruneExpired(c *Client) {
	h.mu.Lock()
	now := time.Now()
	var expired []int64
	for chainID, expiresAt := range c.subs {
		if now.After(expiresAt) {
			delete(c.subs, chainID)
			h.removeRoomLocked(chainRoom(chainID), c)
			expired = append(expired, chainID)
		}
	}
	h.mu.Unlock()
	for _, chainID := range expired {
		c.enqueueControl(control{Action: "unsubscribed", ChainID: chainID, Reason: "ttl"})
	}
}

// PublishRooms delivers msg once to every distinct client in the rooms; slow clients
// are closed rather than blocking the caller.
func (h *Hub) PublishRooms(rooms []string, msg []byte) {
	h.mu.RLock()
	targets := make(map[*Client]struct{})
	for _, room := range rooms {
		for c := range h.rooms[room] {
			targets[c] = struct{}{}
		}
	}
	clients := make([]*Client, 0, len(targets))
	for c := range targets {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.enqueue(msg)
	}
}

// Publish delivers msg to every client in a single room.
func (h *Hub) Publish(room string, msg []byte) {
	h.PublishRooms([]string{room}, msg)
}

// ActiveConnections reports the number of open (or being upgraded) connections.
func (h *Hub) ActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns
}

// ChainSubscriptions reports the total number of chain subscriptions.
func (h *Hub) ChainSubscriptions() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for c := range h.clients {
		total += len(c.subs)
	}
	return total
}

// Close closes every connection with StatusGoingAway. Concurrent calls are safe.
func (h *Hub) Close() {
	h.closeOnce.Do(func() {
		h.mu.RLock()
		clients := make([]*Client, 0, len(h.clients))
		for c := range h.clients {
			clients = append(clients, c)
		}
		h.mu.RUnlock()
		for _, c := range clients {
			c.closeWith(statusGoingAway, "server shutting down")
		}
	})
}
