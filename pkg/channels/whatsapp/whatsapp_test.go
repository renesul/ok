package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/channels"
	"github.com/renesul/ok/pkg/config"
)

func TestMatchJID(t *testing.T) {
	tests := []struct {
		jid     string
		allowed []string
		want    bool
	}{
		{"5511999999999@s.whatsapp.net", []string{"5511999999999"}, true},
		{"5511999999999@s.whatsapp.net", []string{"5511999999999@s.whatsapp.net"}, true},
		{"5511999999999@s.whatsapp.net", []string{"5511888888888"}, false},
		{"120363012345@g.us", []string{"120363012345"}, true},
		{"120363012345@g.us", []string{"120363012345@g.us"}, true},
		{"120363012345@g.us", []string{"999999999"}, false},
		{"5511999999999@s.whatsapp.net", []string{}, false},
		{"5511999999999@s.whatsapp.net", []string{" 5511999999999 "}, true},
	}
	for _, tt := range tests {
		got := matchJID(tt.jid, tt.allowed)
		if got != tt.want {
			t.Errorf("matchJID(%q, %v) = %v, want %v", tt.jid, tt.allowed, got, tt.want)
		}
	}
}

func TestIsAllowedMessage(t *testing.T) {
	messageBus := bus.NewMessageBus()

	t.Run("allowed_groups filters group messages", func(t *testing.T) {
		cfg := config.WhatsAppConfig{
			AllowedGroups:   []string{"120363012345"},
			AllowedContacts: []string{"5511999999999"},
		}
		ch := &WhatsAppChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp", cfg, messageBus, nil),
			config:      cfg,
		}
		sender := bus.SenderInfo{Platform: "whatsapp", PlatformID: "5511888888888@s.whatsapp.net"}

		// Allowed group
		if !ch.isAllowedMessage(types.NewJID("120363012345", types.GroupServer), sender.PlatformID, "120363012345@g.us", sender) {
			t.Error("should allow message from allowed group")
		}
		// Disallowed group
		if ch.isAllowedMessage(types.NewJID("999999999", types.GroupServer), sender.PlatformID, "999999999@g.us", sender) {
			t.Error("should reject message from disallowed group")
		}
	})

	t.Run("allowed_contacts filters direct messages", func(t *testing.T) {
		cfg := config.WhatsAppConfig{
			AllowedContacts: []string{"5511999999999"},
		}
		ch := &WhatsAppChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp", cfg, messageBus, nil),
			config:      cfg,
		}
		sender := bus.SenderInfo{Platform: "whatsapp", PlatformID: "5511999999999@s.whatsapp.net"}

		// Allowed contact
		if !ch.isAllowedMessage(types.NewJID("5511999999999", types.DefaultUserServer), sender.PlatformID, "5511999999999@s.whatsapp.net", sender) {
			t.Error("should allow DM from allowed contact")
		}
		// Disallowed contact
		sender2 := bus.SenderInfo{Platform: "whatsapp", PlatformID: "5511888888888@s.whatsapp.net"}
		if ch.isAllowedMessage(types.NewJID("5511888888888", types.DefaultUserServer), sender2.PlatformID, "5511888888888@s.whatsapp.net", sender2) {
			t.Error("should reject DM from disallowed contact")
		}
	})

	t.Run("no groups configured rejects group messages", func(t *testing.T) {
		cfg := config.WhatsAppConfig{
			AllowedContacts: []string{"5511999999999"},
		}
		ch := &WhatsAppChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp", cfg, messageBus, nil),
			config:      cfg,
		}
		sender := bus.SenderInfo{Platform: "whatsapp", PlatformID: "5511999999999@s.whatsapp.net"}
		if ch.isAllowedMessage(types.NewJID("120363012345", types.GroupServer), sender.PlatformID, "120363012345@g.us", sender) {
			t.Error("should reject group messages when allowed_groups is empty")
		}
	})

	t.Run("empty lists reject everything", func(t *testing.T) {
		cfg := config.WhatsAppConfig{}
		ch := &WhatsAppChannel{
			BaseChannel: channels.NewBaseChannel("whatsapp", cfg, messageBus, nil),
			config:      cfg,
		}
		sender := bus.SenderInfo{Platform: "whatsapp", PlatformID: "anyone"}
		if ch.isAllowedMessage(types.NewJID("anyone", types.DefaultUserServer), sender.PlatformID, "anyone@s.whatsapp.net", sender) {
			t.Error("should reject DMs when allowed_contacts is empty")
		}
		if ch.isAllowedMessage(types.NewJID("somegroup", types.GroupServer), sender.PlatformID, "somegroup@g.us", sender) {
			t.Error("should reject groups when allowed_groups is empty")
		}
	})
}

func TestHandleIncoming_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp", config.WhatsAppConfig{}, messageBus, nil),
		runCtx:      context.Background(),
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: types.NewJID("1001", types.DefaultUserServer),
				Chat:   types.NewJID("1001", types.DefaultUserServer),
			},
			ID:       "mid1",
			PushName: "Alice",
		},
		Message: &waE2E.Message{
			Conversation: proto.String("/help"),
		},
	}

	ch.handleIncoming(evt)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "whatsapp" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/help" {
		t.Fatalf("content=%q", inbound.Content)
	}
}
