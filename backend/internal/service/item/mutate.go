package item

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"
)

// Update is forbidden for ARCHIVED and hard-reserved items.
func (s *Service) Update(ctx context.Context, requesterID uuid.UUID, itemID int64, input UpdateInput) (*entity.Item, error) {
	if err := validator.Validate(&input); err != nil {
		return nil, err
	}

	item, err := s.getOwned(ctx, requesterID, itemID)
	if err != nil {
		return nil, err
	}

	if item.Status == entity.ItemStatusArchived {
		return nil, entity.ErrItemArchived
	}

	if err := s.ensureNoHardReservation(ctx, itemID); err != nil {
		return nil, err
	}

	previousStatus := item.Status

	if input.Title != nil {
		item.Title = strings.TrimSpace(*input.Title)
	}

	if input.Description != nil {
		item.Description = strings.TrimSpace(*input.Description)
	}

	if input.Category != nil {
		item.Category = strings.TrimSpace(*input.Category)
	}

	if input.Status != nil {
		item.Status = *input.Status
	}
	if input.Title != nil || input.Description != nil {
		embeddingValue, err := s.embedItem(ctx, item.Title, item.Description)
		if err != nil {
			return nil, err
		}
		item.Embedding = embeddingValue
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	// emit only on an actual status change; a plain edit would make every client refetch
	if item.Status != previousStatus {
		s.emitItemStatus(ctx, requesterID, item.ID, item.Status)
	}

	return item, nil
}

// Archive moves an item to ARCHIVED. Re-archiving an already archived item is an
// error (entity.ErrItemArchived), not a no-op.
func (s *Service) Archive(ctx context.Context, requesterID uuid.UUID, itemID int64) error {
	item, err := s.getOwned(ctx, requesterID, itemID)
	if err != nil {
		return err
	}

	if item.Status == entity.ItemStatusArchived {
		return entity.ErrItemArchived
	}

	if err := s.ensureNoHardReservation(ctx, itemID); err != nil {
		return err
	}

	if err := s.repo.UpdateStatus(ctx, itemID, entity.ItemStatusArchived); err != nil {
		return err
	}
	s.emitItemStatus(ctx, requesterID, itemID, entity.ItemStatusArchived)
	return nil
}

func (s *Service) emitItemStatus(ctx context.Context, ownerID uuid.UUID, itemID int64, status entity.ItemStatus) {
	if s.emitter == nil {
		return
	}
	s.emitter.Emit(ctx, realtime.Event{
		Type:   realtime.EventItemStatusChanged,
		UserID: ownerID.String(),
		Payload: map[string]any{
			"itemId": itemID,
			"status": string(status),
		},
	})
}

func (s *Service) ensureNoHardReservation(ctx context.Context, itemID int64) error {
	reserved, err := s.repo.HasActiveHardReservation(ctx, itemID)
	if err != nil {
		return err
	}
	if reserved {
		return entity.ErrItemHasHardReservation
	}
	return nil
}
