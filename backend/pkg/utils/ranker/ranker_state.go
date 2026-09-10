package ranker

import (
	"errors"
	"time"
)

// ErrInvalidChainState means Count < 2 or ApprovedVotes outside [0, Count].
var ErrInvalidChainState = errors.New("invalid chain state: Count must be >= 2 and ApprovedVotes must be in [0, Count]")

// ChainStateStatus is the chain lifecycle stage for scoring. It's its own isolated
// enum: Ranker doesn't depend on entity.ChainStatus; on integration an adapter maps
// chain status -> ChainStateStatus.
type ChainStateStatus string

const (
	ChainStateCandidate  ChainStateStatus = "CANDIDATE"
	ChainStateProposed   ChainStateStatus = "PROPOSED"
	ChainStateFrozen     ChainStateStatus = "FROZEN"
	ChainStateInProgress ChainStateStatus = "IN_PROGRESS"
	ChainStateCompleted  ChainStateStatus = "COMPLETED"
	ChainStateBroken     ChainStateStatus = "BROKEN"
)

// StateEvent is which action happened to the chain, used to invoke the matching
// ScoreForX from the embedded matcher and document the recomputation's intent.
type StateEvent string

const (
	EventAdd         StateEvent = "ADD"         // request created/added
	EventDelete      StateEvent = "DELETE"      // request deleted
	EventModify      StateEvent = "MODIFY"      // request changed (= delete + add)
	EventRespond     StateEvent = "RESPOND"     // response (a vote appears)
	EventConfirm     StateEvent = "CONFIRM"     // confirmation
	EventDecline     StateEvent = "DECLINE"     // decline
	EventReplacement StateEvent = "REPLACEMENT" // quick participant replacement
)

// ChainState is a detached chain snapshot + the action that happened + the previous
// score. Ranker recomputes the score fully from this state (no +=/-=): the previous
// score is just an input like length/stage/event, not an accumulated value.
type ChainState struct {
	Count int // chain participant count (Length)

	// Stage is the chain stage (see ChainStateStatus).
	Stage ChainStateStatus
	// Event is the action that triggered the recomputation (see StateEvent).
	Event StateEvent

	// EdgeCosines has one value per A->B edge (cosine similarity, [-1,1]).
	// Length = Count. An empty slice with Count >= 2 is treated as Match = 0 (not an error).
	EdgeCosines []float64

	// ParticipantReliability is each participant's reliability, [0,1].
	// Length = Count. Values <= 0 are replaced with cfg.ReliabilityDefault.
	ParticipantReliability []float64

	// ParticipantClusterSizes is each participant's cluster size (number of member requests).
	// Length = Count. Used for Liquidity. Empty slice -> Liquidity = 0.
	ParticipantClusterSizes []int

	// ApprovedVotes is the count of "approved" votes among participants, [0, Count].
	ApprovedVotes int

	// PrevScore is the previous score (an input to rules/normalization, not a term).
	PrevScore float64

	// Fields below feed the ML head (16 features). FormulaRanker ignores them:
	// weights and the ExtractFeatures subset stay unchanged.

	// Now is the scoring moment. Zero -> time.Now() in ExtractMLFeatures.
	Now time.Time
	// CreatedAt is chains.created_at (for ADD, the draft build time).
	CreatedAt time.Time
	// StageEnteredAt is entry into the current stage (created_at or the latest vote).
	StageEnteredAt time.Time
	// VoteTimes holds the chain's COALESCE(voted_at, created_at) votes, ascending.
	VoteTimes []time.Time
	// OfferedCategories / WantedCategories come from the request→item join, length = Count.
	OfferedCategories []string
	WantedCategories  []string
	// CategoryCounts is the number of ACTIVE items per category (catalog cache).
	CategoryCounts map[string]int
	CategoryTotal  int
}

// ChainFeatures are components extracted from ChainState (all in [0,1]). They feed
// the scoring head: now FormulaRanker, later LightGBM with the same set.
type ChainFeatures struct {
	Match       float64
	Reliability float64
	Liquidity   float64
	Progress    float64
	IsProposed  int // 0 or 1
	IsFrozen    int // 0 or 1
}

// ExtractFeatures converts ChainState into feature components. It validates the
// input (Count >= 2, ApprovedVotes in [0, Count]) and returns ErrInvalidChainState
// on a mismatch.
func ExtractFeatures(s ChainState, cfg RankerConfig) (ChainFeatures, error) {
	cfg = cfg.normalize()

	if s.Count < 2 {
		return ChainFeatures{}, ErrInvalidChainState
	}
	if s.ApprovedVotes < 0 || s.ApprovedVotes > s.Count {
		return ChainFeatures{}, ErrInvalidChainState
	}

	f := ChainFeatures{}

	// Match: mean over edges of (cos+1)/2. Empty edges with Count >= 2 -> 0 (documented).
	if len(s.EdgeCosines) >= 2 {
		var sum float64
		for _, c := range s.EdgeCosines {
			sum += (c + 1) / 2
		}
		f.Match = sum / float64(len(s.EdgeCosines))
	}

	// Reliability: mean participant reliability; values <= 0 -> ReliabilityDefault.
	switch {
	case len(s.ParticipantReliability) == 0:
		f.Reliability = cfg.ReliabilityDefault
	default:
		var sum float64
		for _, r := range s.ParticipantReliability {
			if r <= 0 {
				r = cfg.ReliabilityDefault
			}
			sum += r
		}
		f.Reliability = sum / float64(len(s.ParticipantReliability))
	}

	// Liquidity: min(1, minClusterSize/CAP) saturation. Empty sizes -> 0.
	if len(s.ParticipantClusterSizes) > 0 {
		minSize := s.ParticipantClusterSizes[0]
		for _, sz := range s.ParticipantClusterSizes[1:] {
			if sz < minSize {
				minSize = sz
			}
		}
		f.Liquidity = float64(minSize) / float64(cfg.LiquidityCap)
		if f.Liquidity > 1 {
			f.Liquidity = 1
		}
	}

	// Progress: approvedVotes / Count (Count > 0 is guaranteed by validation).
	if s.Count > 0 {
		f.Progress = float64(s.ApprovedVotes) / float64(s.Count)
	}

	// Stage flags.
	switch s.Stage {
	case ChainStateProposed, ChainStateFrozen, ChainStateInProgress, ChainStateCompleted:
		f.IsProposed = 1
	}
	switch s.Stage {
	case ChainStateFrozen, ChainStateInProgress, ChainStateCompleted:
		f.IsFrozen = 1
	}

	return f, nil
}
