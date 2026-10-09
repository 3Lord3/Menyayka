// Package realtime fans out domain events post-commit to per-user and per-chain
// rooms; the frontend uses them to invalidate queries.
package realtime

import (
	"encoding/json"
	"strconv"
	"time"
)

// Event type contract. Keep in sync with frontend/src/shared/realtime/events.ts.
const (
	EventChainCreated              = "chain.created"
	EventChainStatusChanged        = "chain.status_changed"
	EventChainVoteChanged          = "chain.vote_changed"
	EventChainReplacementAvailable = "chain.replacement_available"
	EventChainReplacementSelected  = "chain.replacement_selected"
	EventChainExpired              = "chain.expired"
	EventChainFulfillmentChanged   = "chain.fulfillment_changed"
	EventChainRemoved              = "chain.removed"
	EventRequestStatusChanged      = "request.status_changed"
	// EventRequestUpdated is an edit that keeps the request status (no "status" in payload).
	EventRequestUpdated    = "request.updated"
	EventItemStatusChanged = "item.status_changed"
)

// Event is the unit of fan-out: ChainID routes to the chain room and participants'
// user rooms, UserID to that user's room.
type Event struct {
	Type    string `json:"type"`
	ChainID int64  `json:"chainId,omitempty"`
	UserID  string `json:"-"`
	// UserIDs lists extra recipients resolvable only at emit time (e.g. a chain being
	// deleted).
	UserIDs []string  `json:"-"`
	Payload any       `json:"payload,omitempty"`
	At      time.Time `json:"at"`
}

// message is the wire representation sent to clients.
type message struct {
	Type    string    `json:"type"`
	ChainID int64     `json:"chainId,omitempty"`
	Payload any       `json:"payload,omitempty"`
	At      time.Time `json:"at"`
}

func marshal(event Event) ([]byte, error) {
	return json.Marshal(message{
		Type:    event.Type,
		ChainID: event.ChainID,
		Payload: event.Payload,
		At:      event.At,
	})
}

func userRoom(userID string) string { return "user:" + userID }
func chainRoom(chainID int64) string {
	return "chain:" + strconv.FormatInt(chainID, 10)
}
