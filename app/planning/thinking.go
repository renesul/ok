package planning

import "ok/app/types"

// Re-export thinking types for convenience.
type ThinkingLevel = types.ThinkingLevel

const (
	ThinkingOff      = types.ThinkingOff
	ThinkingLow      = types.ThinkingLow
	ThinkingMedium   = types.ThinkingMedium
	ThinkingHigh     = types.ThinkingHigh
	ThinkingXHigh    = types.ThinkingXHigh
	ThinkingAdaptive = types.ThinkingAdaptive
)
