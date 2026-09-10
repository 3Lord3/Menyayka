package entity

import "time"

// ExchangeOffer is a user's proposal: which item they give and what they want.
type ExchangeOffer struct {
	ID                int64
	UserID            string
	OfferedItemID     int64
	WantedDescription string
	// WantedCategory is the textual category name of the wanted item (e.g. "Phones").
	WantedCategory string
	WantEmbedding  []float32
	Status         RequestStatus
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExchangeOfferListItem holds an offer and the given item's name, fetched in one
// query without N+1.
type ExchangeOfferListItem struct {
	ExchangeOffer
	OfferedItemTitle string
}
