package routing

import "ok/app/types"

// RuleClassifier is the v1 implementation using weighted structural signals.
type RuleClassifier struct{}

// Score computes the complexity score for the given feature set.
// The returned value is in [0, 1]. Attachments short-circuit to 1.0.
func (c *RuleClassifier) Score(f types.Features) float64 {
	if f.HasAttachments {
		return 1.0
	}

	var score float64

	switch {
	case f.TokenEstimate > 200:
		score += 0.35
	case f.TokenEstimate > 50:
		score += 0.15
	}

	if f.CodeBlockCount > 0 {
		score += 0.40
	}

	switch {
	case f.RecentToolCalls > 3:
		score += 0.25
	case f.RecentToolCalls > 0:
		score += 0.10
	}

	if f.ConversationDepth > 10 {
		score += 0.10
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}
