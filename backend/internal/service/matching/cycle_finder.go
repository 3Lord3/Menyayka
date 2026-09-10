package matching

import (
	"context"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

const (
	minCycleLength        = 2
	maxCycleLength        = 5
	defaultOutgoingK      = 20
	defaultCycleThreshold = 0.5
	defaultDraftCapHint   = 64
)

// FrontierLoader provides batched SQL for CycleFinder (implemented in repository/search).
type FrontierLoader interface {
	LoadOutgoingFrontier(
		ctx context.Context,
		tx database.Tx,
		requestIDs []int64,
		k int,
		threshold float64,
	) ([]entity.CandidateEdge, error)
	LoadIncomingToStart(
		ctx context.Context,
		tx database.Tx,
		startRequestID int64,
		k int,
		threshold float64,
	) ([]entity.CandidateEdge, error)
}

// CycleFinder builds a local graph around a request and finds simple cycles of length 2–5.
type CycleFinder struct {
	loader      FrontierLoader
	outgoingK   int
	maxDrafts   int
	threshold   float64
	minAverage  float64
	maxScoreGap float64
}

type cycleVertex struct {
	clusterID int64
	requestID int64
	ownerID   string
}

func NewCycleFinder(loader FrontierLoader, outgoingK, capHint int, threshold float64) *CycleFinder {
	if outgoingK <= 0 {
		outgoingK = defaultOutgoingK
	}
	if capHint <= 0 {
		capHint = defaultDraftCapHint
	}
	if threshold <= 0 || threshold > 1 {
		threshold = defaultCycleThreshold
	}
	return &CycleFinder{
		loader:      loader,
		outgoingK:   outgoingK,
		maxDrafts:   capHint,
		threshold:   threshold,
		minAverage:  threshold,
		maxScoreGap: 1,
	}
}

// WithQualityRules filters a built cycle (average quality / direction skew).
func (f *CycleFinder) WithQualityRules(minAverage, maxScoreGap float64) *CycleFinder {
	if minAverage > 0 && minAverage <= 1 {
		f.minAverage = minAverage
	}
	if maxScoreGap >= 0 && maxScoreGap <= 1 {
		f.maxScoreGap = maxScoreGap
	}
	return f
}
