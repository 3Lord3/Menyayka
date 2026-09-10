package matching

import (
	"fmt"

	"github.com/Avito-Team-Not-Found/menyayka/pkg/utils/ranker"
)

const (
	RankerModeFormula = "formula"
	RankerModeML      = "ml"
)

// MLRanker computes score via LightGBM. Predict/Extract errors propagate upward
// without falling back to the formula.
type MLRanker struct {
	pred ranker.Predictor
	cfg  ranker.RankerConfig
}

// NewMLRanker requires an already-loaded predictor (fail-fast happens at NewRuntimeRanker).
func NewMLRanker(pred ranker.Predictor) (*MLRanker, error) {
	if pred == nil {
		return nil, fmt.Errorf("ml ranker predictor is nil")
	}
	return &MLRanker{pred: pred, cfg: ranker.NewRankerConfig()}, nil
}

// NewRuntimeRanker picks the head by mode. For mode=ml it loads the model eagerly:
// a corrupt file or feature_names mismatch with FeatureNames is an error, with no
// fallback to the formula.
func NewRuntimeRanker(mode, modelPath string) (ranker.Ranker, error) {
	switch mode {
	case RankerModeFormula:
		return ranker.NewFormulaRanker(ranker.NewRankerConfig()), nil
	case RankerModeML:
		pred, err := ranker.NewLightGBMPredictor(modelPath)
		if err != nil {
			return nil, fmt.Errorf("load lightgbm model: %w", err)
		}
		return NewMLRanker(pred)
	default:
		return nil, fmt.Errorf("unknown ranker mode %q", mode)
	}
}

// Score: ExtractMLFeatures → Predict. It doesn't swallow errors.
func (m *MLRanker) Score(s ranker.ChainState) (float64, error) {
	if m == nil || m.pred == nil {
		return 0, fmt.Errorf("ml ranker is not initialized")
	}
	feats, err := ranker.ExtractMLFeatures(s, m.cfg)
	if err != nil {
		return 0, err
	}
	p, err := m.pred.Predict(feats)
	if err != nil {
		return 0, err
	}
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p, nil
}
