package exchange_offer

import (
	"context"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	offerservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/exchange_offer"
)

// exchangeOfferService is the dependency contract of the exchange-offer HTTP handler.
type exchangeOfferService interface {
	Create(ctx context.Context, userID string, input offerservice.CreateInput) (entity.ExchangeOffer, error)
	Get(ctx context.Context, userID string, requestID int64) (entity.ExchangeOffer, error)
	List(ctx context.Context, userID string) ([]entity.ExchangeOfferListItem, error)
	Update(ctx context.Context, userID string, requestID int64, input offerservice.UpdateInput) (entity.ExchangeOffer, error)
	Delete(ctx context.Context, userID string, requestID, version int64) error
}
