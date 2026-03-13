package routing

import (
	"strings"

	"ok/app/types"
)

// RouteResolver determines which agent handles a message.
// Always routes to the "main" agent.
type RouteResolver struct{}

// NewRouteResolver creates a new route resolver.
// The config parameter is accepted for interface compatibility but not used.
func NewRouteResolver(_ ...any) *RouteResolver {
	return &RouteResolver{}
}

// ResolveRoute determines which agent handles the message and constructs session keys.
// Always returns agent "main" with MatchedBy "default".
func (r *RouteResolver) ResolveRoute(input types.RouteInput) types.ResolvedRoute {
	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	accountID := NormalizeAccountID(input.AccountID)
	peer := input.Peer

	dmScope := types.DMScope(input.DMScope)
	if dmScope == "" {
		dmScope = types.DMScopeMain
	}

	agentID := DefaultAgentID
	sessionKey := strings.ToLower(BuildAgentPeerSessionKey(types.SessionKeyParams{
		AgentID:       agentID,
		Channel:       channel,
		AccountID:     accountID,
		Peer:          peer,
		DMScope:       dmScope,
		IdentityLinks: input.IdentityLinks,
	}))
	mainSessionKey := strings.ToLower(BuildAgentMainSessionKey(agentID))

	return types.ResolvedRoute{
		AgentID:        agentID,
		Channel:        channel,
		AccountID:      accountID,
		SessionKey:     sessionKey,
		MainSessionKey: mainSessionKey,
		MatchedBy:      "default",
	}
}
