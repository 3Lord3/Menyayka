package entity

// RequestStatus describes the exchange request lifecycle stage.
type RequestStatus string

const (
	RequestStatusActive     RequestStatus = "ACTIVE"
	RequestStatusInProposal RequestStatus = "IN_PROPOSAL"
	RequestStatusLocked     RequestStatus = "LOCKED"
	RequestStatusDone       RequestStatus = "DONE"
	RequestStatusInProgress RequestStatus = "IN_PROGRESS"
	RequestStatusRemoved    RequestStatus = "REMOVED"
)
