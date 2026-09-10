package exchange_offer

import (
	"context"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// ExchangeOfferRepository describes the exchange-offer store the service needs.
// Implementations must atomically invalidate candidate chains when a request changes.
type ExchangeOfferRepository interface {
	Create(ctx context.Context, tx database.Tx, request entity.ExchangeOffer) (entity.ExchangeOffer, error)
	Get(ctx context.Context, userID string, requestID int64) (entity.ExchangeOffer, error)
	List(ctx context.Context, userID string) ([]entity.ExchangeOfferListItem, error)
	Update(ctx context.Context, tx database.Tx, request entity.ExchangeOffer, expectedVersion int64) (entity.ExchangeOffer, error)
	Archive(ctx context.Context, tx database.Tx, userID string, requestID, expectedVersion int64) (entity.ExchangeOffer, error)
}

// MatchingFacade describes the part of the matching subsystem the requests service
// needs. The interface lives with its consumer so the matching implementation
// doesn't impose its contract on the rest of the app.
type MatchingFacade interface {
	RebuildForRequest(ctx context.Context, tx database.Tx, requestID int64) ([]entity.ChainDraft, error)
	RemoveRequest(ctx context.Context, tx database.Tx, requestID int64) error
}
