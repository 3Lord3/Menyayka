package chain

import (
	"context"
	"time"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

const FreezeTTL = 24 * time.Hour

// ChainRebuilder rebuilds chains after removing participants from competitors;
// it implements MatchingFacade.RepairAffectedChains.
type ChainRebuilder interface {
	RepairAffectedChains(ctx context.Context, tx database.Tx, affected []int64) error
	RebuildRequests(ctx context.Context, tx database.Tx, requestIDs []int64) error
}

type FreezeService struct {
	store     FreezeStore
	rebuilder ChainRebuilder
}

func NewFreezeService(store FreezeStore, rebuilder ChainRebuilder) *FreezeService {
	return &FreezeService{store: store, rebuilder: rebuilder}
}

func (s *FreezeService) Freeze(ctx context.Context, tx database.Tx, chainID int64) error {
	if s.store == nil {
		return entity.ErrChainRepositoryNotConfigured
	}
	requestIDs, err := s.store.LoadChainRequestIDs(ctx, tx, chainID)
	if err != nil {
		return err
	}
	if err := s.store.LockRequestsForFreeze(ctx, tx, requestIDs); err != nil {
		return err
	}
	if err := s.assertNoDoubleFreeze(ctx, tx, requestIDs); err != nil {
		return err
	}
	deadline := time.Now().Add(FreezeTTL)
	if err := s.store.FreezeChain(ctx, tx, chainID, deadline); err != nil {
		return err
	}
	if err := s.store.LockRequestsInChain(ctx, tx, chainID); err != nil {
		return err
	}
	if err := s.store.MarkItemsUnavailable(ctx, tx, chainID); err != nil {
		return err
	}
	released, err := s.store.ReleaseUnselectedFromChain(ctx, tx, chainID)
	if err != nil {
		return err
	}
	affected, err := s.store.ReleaseCompetitorsFromOtherChains(ctx, tx, chainID)
	if err != nil {
		return err
	}
	if s.rebuilder != nil {
		if err := s.rebuilder.RepairAffectedChains(ctx, tx, affected); err != nil {
			return err
		}
		return s.rebuilder.RebuildRequests(ctx, tx, released)
	}
	return nil
}

// ExpiredChain reports one chain whose deadline elapsed; ReleasedRequestIDs lists
// the requests it freed.
type ExpiredChain struct {
	ChainID            int64
	ReleasedRequestIDs []int64
}

// ExpireResult reports what ExpireDue changed.
type ExpireResult struct {
	ExpiredChains      []ExpiredChain
	ReleasedRequestIDs []int64
}

// ExpireDue applies elapsed chain deadlines in the caller transaction and reports
// the changed chains.
func (s *FreezeService) ExpireDue(ctx context.Context, tx database.Tx) (ExpireResult, error) {
	if s.store == nil {
		return ExpireResult{}, entity.ErrChainRepositoryNotConfigured
	}
	chainIDs, err := s.store.ListExpiredChainIDs(ctx, tx)
	if err != nil {
		return ExpireResult{}, err
	}
	result := ExpireResult{}
	for _, chainID := range chainIDs {
		expired, err := s.store.ExpireProposalIfDue(ctx, tx, chainID)
		if err != nil {
			return ExpireResult{}, err
		}
		if expired {
			result.ExpiredChains = append(result.ExpiredChains, ExpiredChain{ChainID: chainID})
			continue
		}
		requestIDs, expired, err := s.store.ExpireFrozenIfDue(ctx, tx, chainID)
		if err != nil {
			return ExpireResult{}, err
		}
		if expired {
			result.ExpiredChains = append(result.ExpiredChains, ExpiredChain{
				ChainID:            chainID,
				ReleasedRequestIDs: requestIDs,
			})
			result.ReleasedRequestIDs = append(result.ReleasedRequestIDs, requestIDs...)
		}
	}
	if len(result.ReleasedRequestIDs) > 0 && s.rebuilder != nil {
		if err := s.rebuilder.RebuildRequests(ctx, tx, result.ReleasedRequestIDs); err != nil {
			return ExpireResult{}, err
		}
	}
	return result, nil
}

func (s *FreezeService) assertNoDoubleFreeze(ctx context.Context, tx database.Tx, requestIDs []int64) error {
	for _, requestID := range requestIDs {
		status, err := s.store.LoadRequestLiveChainStatus(ctx, tx, requestID)
		if err != nil {
			return err
		}
		if status == entity.ChainStatusFrozen {
			return entity.ErrRequestInTwoFrozenChains
		}
	}
	return nil
}
