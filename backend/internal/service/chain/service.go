package chain

import (
	"time"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/utils/ranker"
)

const (
	minClusters     = 2
	maxClusters     = 5
	confirmationTTL = 24 * time.Hour
	replacementTTL  = 3 * time.Hour
)

// Service persists found chain variants and returns chains available to a user.
type Service struct {
	repository   Repository
	transactions database.TransactionManager
	scorer       ranker.Ranker
	notifier     Notifier
	emitter      realtime.Emitter
	freezer      *FreezeService
}

// NewService creates the chain service.
func NewService(repository Repository, transactions database.TransactionManager) *Service {
	return &Service{repository: repository, transactions: transactions}
}

func (s *Service) WithFreezer(freezer *FreezeService) *Service {
	s.freezer = freezer
	return s
}

// WithScorer hooks up score recomputation on responses/declines.
func (s *Service) WithScorer(scorer ranker.Ranker) *Service {
	s.scorer = scorer
	return s
}

// WithNotifier hooks up sending of chain-event notifications.
func (s *Service) WithNotifier(notifier Notifier) *Service {
	s.notifier = notifier
	return s
}

// WithRealtimeEmitter sets the realtime emitter; pass the same tx-aware emitter
// used as the transaction manager so events flush only after commit.
func (s *Service) WithRealtimeEmitter(emitter realtime.Emitter) *Service {
	s.emitter = emitter
	return s
}

// VoteInput identifies one directed response inside a candidate chain.
type VoteInput struct {
	RequestID       int64 `json:"requestId" validate:"required,gt=0"`
	TargetRequestID int64 `json:"targetRequestId" validate:"required,gt=0,nefield=RequestID"`
}

// FulfillmentResult reports the aggregate state after a handoff or receipt.
type FulfillmentResult struct {
	ChainID   int64              `json:"chainId"`
	RequestID int64              `json:"requestId"`
	Status    entity.ChainStatus `json:"status"`
}
