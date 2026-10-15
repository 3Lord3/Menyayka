package ranker

import "math"

// Ranker computes a barter-chain score. It takes the current state (ChainState) and
// returns a new score in [0, 1]. It's a pure function of state: no +=/-=; the score
// is fully recomputed and therefore doesn't accumulate (a response adds exactly as
// much as withdrawing it removes).
//
// The interface has no context.Context yet; when LightGBM lands we'll add ctx via a
// different head with the same contract, without changing callers or ScoreFor*.
type Ranker interface {
	Score(s ChainState) (float64, error)
}

// ChainScoreCalculator is a Ranker implementation: a linear scoring head over features.
type ChainScoreCalculator struct {
	cfg RankerConfig
}

// FormulaRanker is the linear head's name in spec 3. Weights and feature subset don't change.
type FormulaRanker = ChainScoreCalculator

// NewFormulaRanker is an alias for NewChainScoreCalculator.
func NewFormulaRanker(cfg RankerConfig) *FormulaRanker {
	return NewChainScoreCalculator(cfg)
}

// NewChainScoreCalculator creates a score calculator; cfg is normalized with defaults.
func NewChainScoreCalculator(cfg RankerConfig) *ChainScoreCalculator {
	return &ChainScoreCalculator{cfg: cfg.normalize()}
}

// Score fully recomputes the score from state s and returns it in [0,1].
// Formula (weights sum to 1, all components in [0,1]):
//
//	raw = wM·Match + wR·Reliability + wL·Liquidity + wP·Progress
//	      + wPo·IsProposed + wF·IsFrozen
//
// then clamp to [0,1] and round to 4 decimals.
func (c *ChainScoreCalculator) Score(s ChainState) (float64, error) {
	// PrevScore isn't used as +=/-=; it remains an input to the rules/normalization.
	feats, err := ExtractFeatures(s, c.cfg)
	if err != nil {
		return 0, err
	}

	raw := c.cfg.WeightMatch*feats.Match +
		c.cfg.WeightReliability*feats.Reliability +
		c.cfg.WeightLiquidity*feats.Liquidity +
		c.cfg.WeightProgress*feats.Progress +
		c.cfg.WeightIsProposed*float64(feats.IsProposed) +
		c.cfg.WeightIsFrozen*float64(feats.IsFrozen)

	scaled := raw
	if scaled < 0 {
		scaled = 0
	}
	if scaled > 1 {
		scaled = 1
	}
	return math.Round(scaled*1e4) / 1e4, nil
}

// --- Situational helpers: a convenient facade for other developers. ---
// Each method sets the relevant Event and delegates to the single Score core.
// All are pure: pass the post-action state and get the score.

// ScoreForAdd reflects adding a request to the chain -> recomputes M, R, L.
func (c *ChainScoreCalculator) ScoreForAdd(s ChainState) (float64, error) {
	s.Event = EventAdd
	return c.Score(s)
}

// ScoreForDelete reflects deleting a request -> recomputes M, R, L without it.
// The inverse of Add: pass the state with the participant already removed.
func (c *ChainScoreCalculator) ScoreForDelete(s ChainState) (float64, error) {
	s.Event = EventDelete
	return c.Score(s)
}

// ScoreForModify reflects modifying a request (delete+recreate) -> M, R, L.
func (c *ChainScoreCalculator) ScoreForModify(s ChainState) (float64, error) {
	s.Event = EventModify
	return c.Score(s)
}

// ScoreForRespond reflects a response -> recomputes P (+ is_proposed when the
// chain closes). Pass the state with the increased ApprovedVotes.
func (c *ChainScoreCalculator) ScoreForRespond(s ChainState) (float64, error) {
	s.Event = EventRespond
	return c.Score(s)
}

// ScoreForConfirm reflects a confirmation -> recomputes P (+ is_frozen when fully frozen).
func (c *ChainScoreCalculator) ScoreForConfirm(s ChainState) (float64, error) {
	s.Event = EventConfirm
	return c.Score(s)
}

// ScoreForDecline reflects a decline/withdrawn response -> recomputes P.
// The inverse of Respond: pass the state with the decreased ApprovedVotes.
func (c *ChainScoreCalculator) ScoreForDecline(s ChainState) (float64, error) {
	s.Event = EventDecline
	return c.Score(s)
}

// ScoreForReplacement reflects a quick participant swap (remove one + add another)
// -> recomputes M, R, L with the new participant.
func (c *ChainScoreCalculator) ScoreForReplacement(s ChainState) (float64, error) {
	s.Event = EventReplacement
	return c.Score(s)
}
