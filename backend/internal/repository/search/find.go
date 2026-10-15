package search

import (
	"context"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/repository"
)

func (s *Search) FindSimilarOffers(
	ctx context.Context,
	offer string,
	want string,
	category string,
	wantedCategory string,
	excludeOfferID int64,
	threshold float64,
	directionMargin float64,
	k int,
) ([]entity.Candidate, error) {
	rows, err := s.pool.Query(
		ctx,
		querySimilarOffers,
		offer,
		want,
		excludeOfferID,
		category,
		wantedCategory,
		threshold,
		k,
		directionMargin,
	)
	if err != nil {
		if mappedErr, ok := repository.DBErrToErr(err); ok {
			return nil, mappedErr
		}
		return nil, err
	}
	return collectCandidates(rows)
}
