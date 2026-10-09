package realtime

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/core/config"
	"github.com/Avito-Team-Not-Found/menyayka/internal/middleware"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/codestore"
)

// WSPurpose is the JWT purpose claim of realtime tickets.
const WSPurpose = "ws"

const ticketKeyPrefix = "ws-ticket:"

// TicketIssuer issues and validates one-time realtime tickets.
type TicketIssuer interface {
	GenerateTicket(userID uuid.UUID, purpose string, ttl time.Duration) (string, string, error)
	ParseTicket(tokenStr, wantPurpose string) (uuid.UUID, string, error)
}

// Handler serves the ticket and WebSocket upgrade endpoints.
type Handler struct {
	hub     *Hub
	tokens  TicketIssuer
	tickets *codestore.Store
	access  ChainAccessChecker
	cfg     *config.Config
	logger  *logrus.Logger
}

// NewHandler creates the handler; access gates chain subscriptions on participation.
func NewHandler(hub *Hub, tokens TicketIssuer, tickets *codestore.Store, access ChainAccessChecker, cfg *config.Config, logger *logrus.Logger) *Handler {
	return &Handler{hub: hub, tokens: tokens, tickets: tickets, access: access, cfg: cfg, logger: logger}
}

// Ticket issues a short-lived one-time ticket for the authenticated user.
func (h *Handler) Ticket(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		api.SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return
	}
	ticket, jti, err := h.tokens.GenerateTicket(userID, WSPurpose, h.cfg.WSTicketTTL)
	if err != nil {
		h.logger.WithError(err).Error("failed to generate ws ticket")
		api.SendError(c, http.StatusInternalServerError, "не удалось выдать тикет")
		return
	}
	h.tickets.Save(ticketKeyPrefix+jti, userID.String(), h.cfg.WSTicketTTL)
	api.SendOk(c, http.StatusOK, gin.H{"ticket": ticket})
}

// Connect upgrades to WebSocket, authenticating by burning the one-time ticket
// (Sec-WebSocket-Protocol header).
func (h *Handler) Connect(c *gin.Context) {
	if !h.originAllowed(c.Request) {
		api.SendError(c, http.StatusForbidden, "origin не разрешён")
		return
	}

	ticket := ticketFromRequest(c.Request, h.cfg.WSAllowQueryTicket)
	if ticket == "" {
		api.SendError(c, http.StatusUnauthorized, "тикет отсутствует")
		return
	}

	userID, jti, err := h.tokens.ParseTicket(ticket, WSPurpose)
	if err != nil {
		api.SendError(c, http.StatusUnauthorized, "недействительный тикет")
		return
	}
	stored, ok := h.tickets.Consume(ticketKeyPrefix + jti)
	if !ok || stored != userID.String() {
		api.SendError(c, http.StatusUnauthorized, "тикет уже использован")
		return
	}

	if !h.hub.TryAcquire(userID.String()) {
		api.SendError(c, http.StatusServiceUnavailable, "слишком много соединений")
		return
	}

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		OriginPatterns: h.cfg.WSAllowedOrigins,
		Subprotocols:   []string{"bearer"},
	})
	if err != nil {
		h.hub.ReleaseAcquired(userID.String())
		h.logger.WithError(err).WithField("user_id", userID).Warn("ws upgrade failed")
		return
	}

	client := newClient(h.hub, conn, userID.String(), h.access)
	h.hub.Register(client)
	if h.logger != nil {
		h.logger.WithField("user_id", userID).Debug("realtime connection established")
	}

	ctx, cancel := context.WithCancel(h.hub.ctx)
	defer cancel()
	go client.writePump(ctx)
	go client.heartbeat(ctx)
	client.readPump(ctx)
}

// originAllowed applies a fail-closed policy: an empty whitelist rejects browsers (no
// Origin header is allowed), entries are a full origin or a bare/wildcard hostname on
// any port, and same-origin requests are always allowed.
func (h *Handler) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	if strings.EqualFold(r.Host, parsed.Host) {
		return true
	}
	if len(h.cfg.WSAllowedOrigins) == 0 {
		return false
	}
	for _, pattern := range h.cfg.WSAllowedOrigins {
		if originPatternMatches(pattern, parsed) {
			return true
		}
	}
	return false
}

// originPatternMatches matches one whitelist entry against a parsed origin.
func originPatternMatches(pattern string, origin *url.URL) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "://") {
		// bare hostname matches any scheme and port
		return hostMatches(pattern, strings.ToLower(origin.Hostname()))
	}
	allowed, err := url.Parse(pattern)
	if err != nil || !strings.EqualFold(allowed.Scheme, origin.Scheme) {
		return false
	}
	// an explicit port in the pattern must match
	if allowed.Port() != "" && allowed.Port() != origin.Port() {
		return false
	}
	return hostMatches(strings.ToLower(allowed.Hostname()), strings.ToLower(origin.Hostname()))
}

func hostMatches(pattern, hostname string) bool {
	if pattern == hostname {
		return true
	}
	matched, err := path.Match(pattern, hostname)
	return err == nil && matched
}

// ticketFromRequest prefers the Sec-WebSocket-Protocol subprotocol; the ?ticket=
// query fallback leaks into access logs and is opt-in.
func ticketFromRequest(r *http.Request, allowQueryTicket bool) string {
	for _, header := range r.Header.Values("Sec-WebSocket-Protocol") {
		tokens := strings.Split(header, ",")
		for i, tok := range tokens {
			if strings.EqualFold(strings.TrimSpace(tok), "bearer") && i+1 < len(tokens) {
				return strings.TrimSpace(tokens[i+1])
			}
		}
	}
	if allowQueryTicket {
		return r.URL.Query().Get("ticket")
	}
	return ""
}
