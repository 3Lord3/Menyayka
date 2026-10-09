package chain

import (
	"context"
	"errors"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

func (s *Service) List(ctx context.Context, userID string) ([]entity.Chain, error) {
	if s.repository == nil {
		return nil, entity.ErrChainRepositoryNotConfigured
	}
	if err := s.ExpireDue(ctx); err != nil {
		return nil, err
	}
	return s.repository.List(ctx, userID)
}

// ListForOffer returns the current chains of a request to its owner.

func (s *Service) ListForOffer(ctx context.Context, userID string, offerID int64) ([]entity.Chain, error) {
	if s.repository == nil {
		return nil, entity.ErrChainRepositoryNotConfigured
	}
	if offerID <= 0 {
		return nil, entity.ErrExchangeOfferNotFound
	}
	if err := s.ExpireDue(ctx); err != nil {
		return nil, err
	}
	return s.repository.ListForOffer(ctx, userID, offerID)
}

// Get returns a chain only if the user is one of its participants.

func (s *Service) Get(ctx context.Context, userID string, chainID int64) (entity.Chain, error) {
	if s.repository == nil {
		return entity.Chain{}, entity.ErrChainRepositoryNotConfigured
	}
	if err := s.ExpireDue(ctx); err != nil {
		return entity.Chain{}, err
	}
	chain, err := s.repository.Get(ctx, userID, chainID)
	if !errors.Is(err, entity.ErrChainNotFound) {
		return chain, err
	}
	expired, eventErr := s.repository.HasDeadlineEvent(ctx, userID, chainID)
	if eventErr != nil {
		return entity.Chain{}, eventErr
	}
	if expired {
		return entity.Chain{}, entity.ErrChainConfirmationExpired
	}
	return entity.Chain{}, err
}

func (s *Service) ExpireDue(ctx context.Context) error {
	if s.freezer == nil {
		return nil
	}
	if s.transactions == nil {
		return entity.ErrChainRepositoryNotConfigured
	}
	return s.transactions.WithinTransaction(ctx, func(ctx context.Context, tx database.Tx) error {
		result, err := s.freezer.ExpireDue(ctx, tx)
		if err != nil {
			return err
		}
		// Flushed by the tx-aware emitter after commit: one participant lookup per
		// chain, and only for chains this call actually changed.
		for _, expired := range result.ExpiredChains {
			s.emitExpired(ctx, expired.ChainID, expired.ReleasedRequestIDs)
		}
		return nil
	})
}

// Vote records an idempotent response and atomically proposes a chain when the
// approved responses form a closed cycle through every chain position.
