package types

// RoutePeer represents a chat peer with kind and ID.
type RoutePeer struct {
	Kind string // "direct", "group", "channel"
	ID   string
}

// RouteInput contains the routing context from an inbound message.
type RouteInput struct {
	Channel       string
	AccountID     string
	Peer          *RoutePeer
	DMScope       string              // from config session
	IdentityLinks map[string][]string // from config session
}

// ResolvedRoute is the result of agent routing.
type ResolvedRoute struct {
	AgentID        string
	Channel        string
	AccountID      string
	SessionKey     string
	MainSessionKey string
	MatchedBy      string // "default"
}

// DMScope controls DM session isolation granularity.
type DMScope string

const (
	DMScopeMain                  DMScope = "main"
	DMScopePerPeer               DMScope = "per-peer"
	DMScopePerChannelPeer        DMScope = "per-channel-peer"
	DMScopePerAccountChannelPeer DMScope = "per-account-channel-peer"
)

// SessionKeyParams holds all inputs for session key construction.
type SessionKeyParams struct {
	AgentID       string
	Channel       string
	AccountID     string
	Peer          *RoutePeer
	DMScope       DMScope
	IdentityLinks map[string][]string
}

// ParsedSessionKey is the result of parsing an agent-scoped session key.
type ParsedSessionKey struct {
	AgentID string
	Rest    string
}
