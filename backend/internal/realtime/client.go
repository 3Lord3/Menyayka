package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/sirupsen/logrus"
)

const (
	maxMessageSize = 8 * 1024
	sendBuffer     = 16
)

const (
	statusGoingAway = websocket.StatusGoingAway
	statusTryAgain  = websocket.StatusTryAgainLater
	statusPolicy    = websocket.StatusPolicyViolation
)

// inbound is a client->server application message.
type inbound struct {
	Action  string `json:"action"`
	ChainID int64  `json:"chainId"`
}

// control is a server->client control message.
type control struct {
	Action  string `json:"action"`
	ChainID int64  `json:"chainId,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Client is a single WebSocket connection: readPump runs synchronously, writePump and
// heartbeat in separate goroutines.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	access ChainAccessChecker
	logger *logrus.Logger

	send chan []byte
	subs map[int64]time.Time

	closeOnce sync.Once
	// closeFunc is a field so tests can observe drops without a real socket.
	closeFunc func(code websocket.StatusCode, reason string)
}

func newClient(hub *Hub, conn *websocket.Conn, userID string, access ChainAccessChecker) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		userID: userID,
		access: access,
		logger: hub.logger,
		send:   make(chan []byte, sendBuffer),
		subs:   make(map[int64]time.Time),
	}
}

// enqueue pushes without blocking; a full buffer drops the slow client.
func (c *Client) enqueue(msg []byte) {
	select {
	case c.send <- msg:
	default:
		if c.logger != nil {
			c.logger.WithField("user_id", c.userID).Warn("realtime client buffer overflow, closing connection")
		}
		c.dropAsync(statusTryAgain, "slow consumer")
	}
}

func (c *Client) enqueueControl(msg control) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.enqueue(data)
}

func (c *Client) closeWith(code websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		if c.closeFunc != nil {
			c.closeFunc(code, reason)
			return
		}
		// coder/websocket bounds the close handshake internally (5s each).
		_ = c.conn.Close(code, reason)
	})
}

// dropAsync closes without blocking the caller: a stalled peer's close handshake must
// not stall the event fan-out.
func (c *Client) dropAsync(code websocket.StatusCode, reason string) {
	go c.closeWith(code, reason)
}

// readPump runs the message loop until the connection closes.
func (c *Client) readPump(ctx context.Context) {
	defer c.hub.Unregister(c)
	c.conn.SetReadLimit(maxMessageSize)
	limiter := newRateLimiter(c.hub.cfg.MsgRate, c.hub.cfg.MsgRateWindow)
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		if !limiter.allow() {
			c.enqueueControl(control{Action: "error", Code: "rate_limited", Message: "too many messages"})
			if limiter.exceeded() {
				c.closeWith(statusPolicy, "message rate exceeded")
				return
			}
			continue
		}
		// Accepted frames extend the subscription TTL; server Ping doesn't.
		c.hub.Touch(c)

		var msg inbound
		if err := json.Unmarshal(data, &msg); err != nil {
			c.enqueueControl(control{Action: "error", Code: "bad_request", Message: "invalid message"})
			continue
		}
		c.handle(ctx, msg)
	}
}

func (c *Client) handle(ctx context.Context, msg inbound) {
	switch msg.Action {
	case "subscribe":
		if msg.ChainID <= 0 {
			c.enqueueControl(control{Action: "error", Code: "bad_request", Message: "chainId is required"})
			return
		}
		// Enforce the same rule as the REST read path before fan-out.
		if !c.canSubscribe(ctx, msg.ChainID) {
			return
		}
		if !c.hub.Subscribe(c, msg.ChainID) {
			c.enqueueControl(control{Action: "error", Code: "sub_limit", Message: "subscription limit reached"})
			return
		}
		c.enqueueControl(control{Action: "subscribed", ChainID: msg.ChainID})
	case "unsubscribe":
		if msg.ChainID <= 0 {
			c.enqueueControl(control{Action: "error", Code: "bad_request", Message: "chainId is required"})
			return
		}
		c.hub.Unsubscribe(c, msg.ChainID)
		c.enqueueControl(control{Action: "unsubscribed", ChainID: msg.ChainID})
	case "ping":
		c.enqueueControl(control{Action: "pong"})
	default:
		c.enqueueControl(control{Action: "error", Code: "unknown_action", Message: "unknown action"})
	}
}

// canSubscribe checks membership, sending an error otherwise. Fail-closed; the error
// does not distinguish a missing chain from a foreign one.
func (c *Client) canSubscribe(ctx context.Context, chainID int64) bool {
	if c.access == nil {
		c.enqueueControl(control{Action: "error", ChainID: chainID, Code: "internal", Message: "chain access is not configured"})
		return false
	}
	allowed, err := c.access.IsChainParticipant(ctx, c.userID, chainID)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":  c.userID,
				"chain_id": chainID,
			}).Warn("failed to check chain participation")
		}
		c.enqueueControl(control{Action: "error", ChainID: chainID, Code: "internal", Message: "cannot verify chain access"})
		return false
	}
	if !allowed {
		c.enqueueControl(control{Action: "error", ChainID: chainID, Code: "forbidden", Message: "chain is not accessible"})
		return false
	}
	return true
}

// writePump writes queued messages until the context is cancelled.
func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, c.hub.cfg.WriteTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// heartbeat pings the peer and prunes expired subscriptions.
func (c *Client) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(c.hub.cfg.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, c.hub.cfg.Heartbeat)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				c.closeWith(statusGoingAway, "ping timeout")
				return
			}
			c.hub.pruneExpired(c)
		}
	}
}

// rateLimiter is a fixed-window per-client message counter.
type rateLimiter struct {
	max        int
	window     time.Duration
	start      time.Time
	count      int
	violations int
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{max: max, window: window, start: time.Now()}
}

func (r *rateLimiter) allow() bool {
	if r.max <= 0 {
		return true
	}
	now := time.Now()
	if now.Sub(r.start) >= r.window {
		r.start = now
		r.count = 0
	}
	if r.count >= r.max {
		r.violations++
		return false
	}
	r.count++
	return true
}

// exceeded reports persistent flooding past the warnings.
func (r *rateLimiter) exceeded() bool {
	return r.violations > 5
}
