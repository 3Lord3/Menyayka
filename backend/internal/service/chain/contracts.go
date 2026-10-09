package chain

import (
	"context"
	"time"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/utils/ranker"
)

// CandidateStore persists found candidate chains; SaveCandidates returns the IDs
// actually inserted.
type CandidateStore interface {
	SaveCandidates(ctx context.Context, tx database.Tx, drafts []entity.ChainDraft) ([]int64, error)
}

// ChainReader reads chains for a participant.
type ChainReader interface {
	List(ctx context.Context, userID string) ([]entity.Chain, error)
	ListForOffer(ctx context.Context, userID string, offerID int64) ([]entity.Chain, error)
	Get(ctx context.Context, userID string, chainID int64) (entity.Chain, error)
	HasDeadlineEvent(ctx context.Context, userID string, chainID int64) (bool, error)
}

// VoteStore handles responses at the CANDIDATE stage and the transition to PROPOSED.
type VoteStore interface {
	LockForVote(ctx context.Context, tx database.Tx, chainID int64) (entity.ChainStatus, int, error)
	ValidateVoteParticipants(ctx context.Context, tx database.Tx, userID string, chainID, requestID, targetRequestID int64, chainLength int) error
	GetVote(ctx context.Context, tx database.Tx, userID string, chainID, requestID, targetRequestID int64) (entity.ChainVote, error)
	UpsertPendingVote(ctx context.Context, tx database.Tx, chainID, requestID, targetRequestID int64) (time.Time, error)
	DeletePendingVote(ctx context.Context, tx database.Tx, chainID, requestID, targetRequestID int64) error
	ListPendingVoteEdges(ctx context.Context, tx database.Tx, chainID int64) ([]entity.VoteEdge, error)
	Propose(ctx context.Context, tx database.Tx, chainID int64, requestIDsByPosition []int64, confirmationDeadline time.Time) error
	MarkRequestInProposal(ctx context.Context, tx database.Tx, requestID int64) error
	RestoreActiveIfNoPendingVotes(ctx context.Context, tx database.Tx, requestID int64) error
}

// ProposalStore handles the confirmation round (confirm/think/decline/unconfirm)
// and proposal deadlines.
type ProposalStore interface {
	ExpireProposalIfDue(ctx context.Context, tx database.Tx, chainID int64) (bool, error)
	ConfirmParticipant(ctx context.Context, tx database.Tx, chainID, requestID, targetRequestID int64) error
	UnconfirmParticipant(ctx context.Context, tx database.Tx, chainID, requestID, targetRequestID int64) error
	MarkParticipantThinking(ctx context.Context, tx database.Tx, chainID, requestID, targetRequestID int64) error
	DeclineParticipant(ctx context.Context, tx database.Tx, chainID, requestID int64, fastReplacementEligible bool) (bool, entity.ChainStatus, error)
	PrepareFrozenReplacement(ctx context.Context, tx database.Tx, chainID int64, deadline time.Time) error
	IsFrozenReplacement(ctx context.Context, tx database.Tx, chainID int64) (bool, error)
	FindParticipantEdge(ctx context.Context, tx database.Tx, chainID int64, userID string) (requestID, targetRequestID int64, err error)
	CountApprovedVoters(ctx context.Context, tx database.Tx, chainID int64) (int, error)
	CountApprovedVotersExcept(ctx context.Context, tx database.Tx, chainID, requestID int64) (int, error)
	MarkRequestLocked(ctx context.Context, tx database.Tx, requestID int64) error
	LockRequestsForFreeze(ctx context.Context, tx database.Tx, requestIDs []int64) error
	LoadChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
}

// FreezeStore handles chain freezing, competitor removal, and overdue deadlines.
type FreezeStore interface {
	FreezeChain(ctx context.Context, tx database.Tx, chainID int64, deadline time.Time) error
	LockRequestsInChain(ctx context.Context, tx database.Tx, chainID int64) error
	MarkItemsUnavailable(ctx context.Context, tx database.Tx, chainID int64) error
	LoadChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
	LockRequestsForFreeze(ctx context.Context, tx database.Tx, requestIDs []int64) error
	LoadRequestLiveChainStatus(ctx context.Context, tx database.Tx, requestID int64) (entity.ChainStatus, error)
	ReleaseUnselectedFromChain(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
	ReleaseCompetitorsFromOtherChains(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
	ListExpiredChainIDs(ctx context.Context, tx database.Tx) ([]int64, error)
	ExpireProposalIfDue(ctx context.Context, tx database.Tx, chainID int64) (bool, error)
	ExpireFrozenIfDue(ctx context.Context, tx database.Tx, chainID int64) ([]int64, bool, error)
}

// FulfillmentStore — handoff / receipt / completion.
type FulfillmentStore interface {
	LockForVote(ctx context.Context, tx database.Tx, chainID int64) (entity.ChainStatus, int, error)
	MarkRequestInProgress(ctx context.Context, tx database.Tx, chainID, requestID int64) (entity.RequestStatus, error)
	StartChain(ctx context.Context, tx database.Tx, chainID int64) error
	FindReceiptRequestStatus(ctx context.Context, tx database.Tx, chainID, requestID int64, userID string) (entity.RequestStatus, error)
	MarkRequestDone(ctx context.Context, tx database.Tx, requestID int64) error
	AllChainRequestsDone(ctx context.Context, tx database.Tx, chainID int64) (bool, error)
	CompleteChain(ctx context.Context, tx database.Tx, chainID int64) error
}

// ReplacementStore handles a quick participant replacement.
type ReplacementStore interface {
	ListReplacementOptions(ctx context.Context, userID string, chainID int64) ([]entity.ReplacementOption, error)
	SelectReplacement(ctx context.Context, tx database.Tx, userID string, chainID, replacementRequestID int64) error
	LockForVote(ctx context.Context, tx database.Tx, chainID int64) (entity.ChainStatus, int, error)
	ExpireProposalIfDue(ctx context.Context, tx database.Tx, chainID int64) (bool, error)
}

// ScoreStore holds features and writes score / ML-ranker context.
type ScoreStore interface {
	LoadScoreFeatures(ctx context.Context, tx database.Tx, chainID int64) (cosines []float64, reliability []float64, sizes []int, err error)
	LoadRankerContext(ctx context.Context, tx database.Tx, chainID int64) (ranker.ContextSnapshot, error)
	LoadRankerContextForRequests(ctx context.Context, tx database.Tx, requestIDs []int64) (ranker.ContextSnapshot, error)
	CountPendingVoters(ctx context.Context, tx database.Tx, chainID int64) (int, error)
	CountApprovedVoters(ctx context.Context, tx database.Tx, chainID int64) (int, error)
	UpdateScore(ctx context.Context, tx database.Tx, chainID int64, score float64) error
}

// LifecycleStore supports rebuild/deletion for the matcher.
type LifecycleStore interface {
	ListChainsContainingRequest(ctx context.Context, tx database.Tx, requestID int64) ([]int64, error)
	DeleteRequestParticipation(ctx context.Context, tx database.Tx, requestID int64) error
	DeleteChain(ctx context.Context, tx database.Tx, chainID int64) error
	LoadChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
	LoadActiveChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error)
}

// ParticipantReader resolves chain participants for the realtime fan-out.
type ParticipantReader interface {
	ListChainParticipantUserIDs(ctx context.Context, chainID int64) ([]string, error)
}

// Repository composes the store interfaces for production wiring (*Postgres).
// Prefer the narrow interfaces above for new dependencies.
type Repository interface {
	CandidateStore
	ChainReader
	VoteStore
	ProposalStore
	FreezeStore
	FulfillmentStore
	ReplacementStore
	ScoreStore
	LifecycleStore
	ParticipantReader
}

// Notifier sends optional chain-event notifications.
type Notifier interface {
	NotifyChainFrozen(ctx context.Context, chainID int64) error
	NotifyReplacementInvited(ctx context.Context, chainID, requestID int64) error
}
