package routing

import (
	"testing"

	"ok/app/types"
)

func TestResolveRoute_DefaultAgent(t *testing.T) {
	r := NewRouteResolver()

	route := r.ResolveRoute(types.RouteInput{
		Channel: "telegram",
		Peer:    &types.RoutePeer{Kind: "direct", ID: "user1"},
		DMScope: "per-peer",
	})

	if route.AgentID != DefaultAgentID {
		t.Errorf("AgentID = %q, want %q", route.AgentID, DefaultAgentID)
	}
	if route.MatchedBy != "default" {
		t.Errorf("MatchedBy = %q, want 'default'", route.MatchedBy)
	}
}

func TestResolveRoute_SessionKey(t *testing.T) {
	r := NewRouteResolver()

	route := r.ResolveRoute(types.RouteInput{
		Channel: "telegram",
		Peer:    &types.RoutePeer{Kind: "direct", ID: "user1"},
		DMScope: "per-peer",
	})

	if route.SessionKey == "" {
		t.Error("SessionKey should not be empty")
	}
	if route.MainSessionKey == "" {
		t.Error("MainSessionKey should not be empty")
	}
}

func TestResolveRoute_ChannelNormalization(t *testing.T) {
	r := NewRouteResolver()

	route := r.ResolveRoute(types.RouteInput{
		Channel: "  Telegram  ",
		DMScope: "per-peer",
	})

	if route.Channel != "telegram" {
		t.Errorf("Channel = %q, want 'telegram'", route.Channel)
	}
}

func TestResolveRoute_AlwaysDefault(t *testing.T) {
	r := NewRouteResolver()

	// Even with various inputs, always returns "main" with "default"
	inputs := []types.RouteInput{
		{Channel: "telegram", Peer: &types.RoutePeer{Kind: "direct", ID: "user1"}},
		{Channel: "discord", Peer: &types.RoutePeer{Kind: "channel", ID: "ch1"}},
		{Channel: "slack", AccountID: "bot1"},
		{Channel: "cli"},
	}

	for _, input := range inputs {
		route := r.ResolveRoute(input)
		if route.AgentID != DefaultAgentID {
			t.Errorf("AgentID = %q for channel %q, want %q", route.AgentID, input.Channel, DefaultAgentID)
		}
		if route.MatchedBy != "default" {
			t.Errorf("MatchedBy = %q for channel %q, want 'default'", route.MatchedBy, input.Channel)
		}
	}
}
