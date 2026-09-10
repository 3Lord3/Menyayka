package entity

// ChainDraft is a not-yet-persisted chain; Participant order equals cycle positions.
// ClusterSizes/EdgeCosines/ParticipantReliability are raw features; score is set by
// the facade's ranker.
type ChainDraft struct {
	Participants           []ChainDraftParticipant
	ClusterSizes           []int
	EdgeCosines            []float64
	ParticipantReliability []float64
	Score                  float64
}

// RequestID represents the cluster entry; vertex identity is ClusterID.
type ChainDraftParticipant struct {
	ClusterID int64
	RequestID int64
}
