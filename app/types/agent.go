package types

import (
	"ok/providers"
)

// Features holds the structural signals extracted from a message and its session context.
type Features struct {
	TokenEstimate     int
	CodeBlockCount    int
	RecentToolCalls   int
	ConversationDepth int
	HasAttachments    bool
}

// Classifier evaluates a feature set and returns a complexity score in [0, 1].
type Classifier interface {
	Score(f Features) float64
}

// ModelRouter selects the appropriate model tier for each incoming message.
type ModelRouter interface {
	SelectModel(msg string, history []providers.Message, primaryModel string) (model string, usedLight bool, score float64)
	LightModel() string
	Threshold() float64
}
