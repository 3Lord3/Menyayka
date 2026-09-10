package chain

import (
	"context"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// ListChainsContainingRequest returns chains involving the request.
func (s *Service) ListChainsContainingRequest(ctx context.Context, tx database.Tx, requestID int64) ([]int64, error) {
	if s.repository == nil {
		return nil, entity.ErrChainRepositoryNotConfigured
	}
	return s.repository.ListChainsContainingRequest(ctx, tx, requestID)
}

// DeleteRequestParticipation removes a request's participation in chains and its votes.
func (s *Service) DeleteRequestParticipation(ctx context.Context, tx database.Tx, requestID int64) error {
	if s.repository == nil {
		return entity.ErrChainRepositoryNotConfigured
	}
	return s.repository.DeleteRequestParticipation(ctx, tx, requestID)
}

// DeleteChain deletes a chain entirely via cascade.
func (s *Service) DeleteChain(ctx context.Context, tx database.Tx, chainID int64) error {
	if s.repository == nil {
		return entity.ErrChainRepositoryNotConfigured
	}
	return s.repository.DeleteChain(ctx, tx, chainID)
}

// LoadChainRequestIDs returns the chain's participant requests.
func (s *Service) LoadChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error) {
	if s.repository == nil {
		return nil, entity.ErrChainRepositoryNotConfigured
	}
	return s.repository.LoadChainRequestIDs(ctx, tx, chainID)
}

func (s *Service) LoadActiveChainRequestIDs(ctx context.Context, tx database.Tx, chainID int64) ([]int64, error) {
	if s.repository == nil {
		return nil, entity.ErrChainRepositoryNotConfigured
	}
	return s.repository.LoadActiveChainRequestIDs(ctx, tx, chainID)
}
