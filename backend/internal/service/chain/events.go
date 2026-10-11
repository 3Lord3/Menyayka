package chain

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
)

// chainEventPayload is the minimal chain-event payload; the frontend only uses it to
// invalidate queries and refetches the body over REST.
type chainEventPayload struct {
	ChainID              int64   `json:"chainId"`
	Status               string  `json:"status,omitempty"`
	RequestID            int64   `json:"requestId,omitempty"`
	TargetRequestID      int64   `json:"targetRequestId,omitempty"`
	ReplacementRequestID int64   `json:"replacementRequestId,omitempty"`
	ReleasedRequestIDs   []int64 `json:"releasedRequestIds,omitempty"`
}

func (s *Service) emit(ctx context.Context, eventType string, payload chainEventPayload) {
	s.emitTo(ctx, eventType, payload, nil)
}

// emitTo emits with explicit recipients, needed when participants cannot be resolved
// after commit (a chain being deleted).
func (s *Service) emitTo(ctx context.Context, eventType string, payload chainEventPayload, userIDs []string) {
	if s.emitter == nil {
		return
	}
	s.emitter.Emit(ctx, realtime.Event{
		Type:    eventType,
		ChainID: payload.ChainID,
		UserIDs: userIDs,
		Payload: payload,
	})
}

// emitExpired reports a chain that just rolled back from an overdue deadline.
func (s *Service) emitExpired(ctx context.Context, chainID int64, releasedRequestIDs []int64) {
	s.emit(ctx, realtime.EventChainExpired, chainEventPayload{
		ChainID:            chainID,
		ReleasedRequestIDs: releasedRequestIDs,
	})
	s.emit(ctx, realtime.EventChainStatusChanged, chainEventPayload{
		ChainID: chainID,
		Status:  string(entity.ChainStatusCandidate),
	})
}

// participantUserIDs resolves participants eagerly, before the delete removes the rows.
func (s *Service) participantUserIDs(ctx context.Context, chainID int64) []string {
	if s.repository == nil {
		return nil
	}
	userIDs, err := s.repository.ListChainParticipantUserIDs(ctx, chainID)
	if err != nil {
		logrus.WithError(err).WithField("chain_id", chainID).
			Warn("failed to resolve chain participants for the realtime event")
		return nil
	}
	return userIDs
}
