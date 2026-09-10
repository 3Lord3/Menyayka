package ranker

// RankerConfig holds the FormulaRanker linear-head parameters. Component weights sum
// to 1; a 0 (or negative) value falls back to the default.
//
// LiquidityCap and ReliabilityDefault are reused by ExtractMLFeatures.
// The spec-3 linear-head weights don't change.
type RankerConfig struct {
	WeightMatch       float64 // Match weight        (default 0.35)
	WeightReliability float64 // Reliability weight  (default 0.20)
	WeightLiquidity   float64 // Liquidity weight    (default 0.10)
	WeightProgress    float64 // Progress weight     (default 0.15)
	WeightIsProposed  float64 // is_proposed weight  (default 0.10)
	WeightIsFrozen    float64 // is_frozen weight    (default 0.10)

	LiquidityCap       int     // Liquidity saturation cap: min(1, minCluster/CAP) (default 4)
	ReliabilityDefault float64 // participant reliability when not provided (default 0.75)
}

// NewRankerConfig returns a config with default values.
// Score is in [0,1]; formula (weights sum to 1):
//
//	score = wM·M + wR·R + wL·L + wP·P + wPo·is_proposed + wF·is_frozen
func NewRankerConfig() RankerConfig {
	return RankerConfig{
		WeightMatch:       0.35,
		WeightReliability: 0.20,
		WeightLiquidity:   0.10,
		WeightProgress:    0.15,
		WeightIsProposed:  0.10,
		WeightIsFrozen:    0.10,

		LiquidityCap:       4,
		ReliabilityDefault: 0.75,
	}
}

// normalize fills zero/negative fields with defaults.
func (c RankerConfig) normalize() RankerConfig {
	d := NewRankerConfig()
	if c.WeightMatch <= 0 {
		c.WeightMatch = d.WeightMatch
	}
	if c.WeightReliability <= 0 {
		c.WeightReliability = d.WeightReliability
	}
	if c.WeightLiquidity <= 0 {
		c.WeightLiquidity = d.WeightLiquidity
	}
	if c.WeightProgress <= 0 {
		c.WeightProgress = d.WeightProgress
	}
	if c.WeightIsProposed <= 0 {
		c.WeightIsProposed = d.WeightIsProposed
	}
	if c.WeightIsFrozen <= 0 {
		c.WeightIsFrozen = d.WeightIsFrozen
	}
	if c.LiquidityCap <= 0 {
		c.LiquidityCap = d.LiquidityCap
	}
	if c.ReliabilityDefault <= 0 {
		c.ReliabilityDefault = d.ReliabilityDefault
	}
	return c
}
