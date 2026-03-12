package routing

import (
	"ok/app/types"
	"ok/internal/logger"
	"ok/providers"
)

// defaultThreshold is used when the config threshold is zero or negative.
const defaultThreshold = 0.35

// RouterConfig holds the validated model routing settings.
type RouterConfig struct {
	LightModel string
	Threshold  float64
}

// Router selects the appropriate model tier for each incoming message.
// It is safe for concurrent use from multiple goroutines.
type Router struct {
	cfg        RouterConfig
	classifier types.Classifier
}

// NewRouter creates a Router with the given config and the default RuleClassifier.
func NewRouter(cfg RouterConfig) *Router {
	if cfg.Threshold <= 0 {
		cfg.Threshold = defaultThreshold
	}
	return &Router{
		cfg:        cfg,
		classifier: &RuleClassifier{},
	}
}

// NewRouterWithClassifier creates a Router with a custom Classifier.
// Intended for unit tests that need to inject a deterministic scorer.
func NewRouterWithClassifier(cfg RouterConfig, c types.Classifier) *Router {
	if cfg.Threshold <= 0 {
		cfg.Threshold = defaultThreshold
	}
	return &Router{cfg: cfg, classifier: c}
}

// SelectModel returns the model to use for this conversation turn.
func (r *Router) SelectModel(
	msg string,
	history []providers.Message,
	primaryModel string,
) (model string, usedLight bool, score float64) {
	features := ExtractFeatures(msg, history)
	score = r.classifier.Score(features)
	if score < r.cfg.Threshold {
		logger.InfoCF("routing", "Model selected", map[string]any{
			"model":     r.cfg.LightModel,
			"tier":      "light",
			"score":     score,
			"threshold": r.cfg.Threshold,
		})
		return r.cfg.LightModel, true, score
	}
	logger.InfoCF("routing", "Model selected", map[string]any{
		"model":     primaryModel,
		"tier":      "primary",
		"score":     score,
		"threshold": r.cfg.Threshold,
	})
	return primaryModel, false, score
}

// LightModel returns the configured light model name.
func (r *Router) LightModel() string {
	return r.cfg.LightModel
}

// Threshold returns the complexity threshold in use.
func (r *Router) Threshold() float64 {
	return r.cfg.Threshold
}
