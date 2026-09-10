package entity

import "time"

// ChainStatus describes the exchange chain lifecycle stage.
type ChainStatus string

const (
	ChainStatusCandidate  ChainStatus = "CANDIDATE"
	ChainStatusProposed   ChainStatus = "PROPOSED"
	ChainStatusFrozen     ChainStatus = "FROZEN"
	ChainStatusInProgress ChainStatus = "IN_PROGRESS"
	ChainStatusCompleted  ChainStatus = "COMPLETED"
	ChainStatusBroken     ChainStatus = "BROKEN"
)

// Chain holds a sequence of clusters and the available requests in each position.
type Chain struct {
	ID               int64
	Status           ChainStatus
	Score            float64
	Length           int
	FreezeDeadlineAt *time.Time
	InvalidReason    *string
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CurrentRequestID int64
	CurrentPosition  int
	Participants     []ChainParticipant
}

// ChainParticipant describes an available request in a cluster at a cycle position.
// A position may hold several requests of one cluster. Each participant receives
// the item from the next position and gives theirs to the previous one around the cycle.
type ChainParticipant struct {
	ID                     int64
	ChainID                int64
	ClusterID              int64
	RequestID              int64
	Position               int
	OwnerUserID            string
	OfferedItemID          int64
	OfferedItemTitle       string
	OfferedItemDescription string
	WantedDescription      string
	ImageURL               *string
	RequestStatus          RequestStatus
	Vote                   *VoteValue
	CreatedAt              time.Time
}

// ReplacementOption is an ACTIVE request from the declining cluster; a swap without
// rebuilding the cycle.
type ReplacementOption struct {
	RequestID         int64
	OfferedItemID     int64
	Title             string
	Description       string
	WantedDescription string
	ImageURL          *string
	Reliability       float64
	RespondedAt       time.Time
}
