package entity

// Candidate is a lightweight vector-search hit (Top-K or by threshold): a raw
// pgvector find of what may match by semantic similarity. Full request/item data
// is loaded separately when needed (avoids N+1).
type Candidate struct {
	// RequestID is the owner request of the found "receive", or the found request
	// itself for the "give" direction. Key for further business matching.
	RequestID int64
	// ItemID is the candidate item (for "receive" it's another user's item that can
	// be given; for "give" it's the item the request is based on).
	ItemID int64
	// OwnerID is the item/request owner, used to quickly exclude own records and
	// check availability in the business layer.
	OwnerID string
	// Score is cosine similarity in [-1, 1], where 1 is a perfect match. Computed in
	// SQL as 1 - (embedding <=> want).
	Score float64
}

// CandidateEdge is a directed edge of the local exchange graph. From wants To's
// item; both ends carry a cluster_id so DFS can't include a cluster twice.
type CandidateEdge struct {
	FromRequestID int64
	FromClusterID int64
	FromOwnerID   string
	ToRequestID   int64
	ToClusterID   int64
	ToOwnerID     string
	Score         float64
}
