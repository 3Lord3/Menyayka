package entity

import (
	"time"

	"github.com/google/uuid"
)

// ItemStatus describes the item lifecycle stage.
type ItemStatus string

const (
	ItemStatusActive      ItemStatus = "ACTIVE"
	ItemStatusUnavailable ItemStatus = "UNAVAILABLE"
	ItemStatusArchived    ItemStatus = "ARCHIVED"
)

// Item is a product a user can offer in an exchange request.
type Item struct {
	ID          int64
	OwnerUserID uuid.UUID
	Title       string
	Description string
	// Category is the textual category name (e.g. "Phones").
	Category  string
	Embedding []float32
	// ImageURL is the public object-storage (MinIO) photo URL; nil while no photo is uploaded.
	ImageURL  *string
	Status    ItemStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
