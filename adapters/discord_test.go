package adapters

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func newTestDiscord(runner AgentRunner, ownerID string) *DiscordAdapter {
	return NewDiscordAdapter(runner, "fake-token", ownerID, zap.NewNop())
}

func discordSession(botID string) *discordgo.Session {
	return &discordgo.Session{
		State: &discordgo.State{
			Ready: discordgo.Ready{
				User: &discordgo.User{ID: botID},
			},
		},
	}
}

func discordMsgCreate(authorID, content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:  &discordgo.User{ID: authorID, Username: "testuser"},
			Content: content,
		},
	}
}

// --- Enabled ---

func TestDiscordAdapter_Enabled(t *testing.T) {
	tests := []struct {
		token, ownerID string
		want           bool
	}{
		{"tok", "owner1", true},
		{"", "owner1", false},
		{"tok", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		a := NewDiscordAdapter(newMockRunner(), tt.token, tt.ownerID, zap.NewNop())
		if got := a.Enabled(); got != tt.want {
			t.Errorf("Enabled(%q, %q) = %v, want %v", tt.token, tt.ownerID, got, tt.want)
		}
	}
}

// --- handleMessage ---

func TestDiscordAdapter_HandleMessage_IgnoresBotSelf(t *testing.T) {
	mock := newMockRunner()
	a := newTestDiscord(mock, "owner1")
	s := discordSession("bot123")
	a.handleMessage(s, discordMsgCreate("bot123", "hi"))
	if mock.called {
		t.Error("expected Run() NOT called for bot self-message")
	}
}

func TestDiscordAdapter_HandleMessage_IgnoresNonOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestDiscord(mock, "owner1")
	s := discordSession("bot123")
	a.handleMessage(s, discordMsgCreate("stranger", "hi"))
	if mock.called {
		t.Error("expected Run() NOT called for non-owner")
	}
}

func TestDiscordAdapter_HandleMessage_IgnoresEmptyContent(t *testing.T) {
	mock := newMockRunner()
	a := newTestDiscord(mock, "owner1")
	s := discordSession("bot123")
	a.handleMessage(s, discordMsgCreate("owner1", ""))
	if mock.called {
		t.Error("expected Run() NOT called for empty content")
	}
}

func TestDiscordAdapter_HandleMessage_ProcessesOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestDiscord(mock, "owner1")
	s := discordSession("bot123")
	a.handleMessage(s, discordMsgCreate("owner1", "deploy prod"))
	if !mock.called {
		t.Fatal("expected Run() to be called")
	}
	if mock.lastInput != "deploy prod" {
		t.Errorf("Run() input = %q, want %q", mock.lastInput, "deploy prod")
	}
}
