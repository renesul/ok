package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"

	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func TestHandleMessage_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := events.NewMessageBus()
	cfg := &config.Config{
		Channels: config.ChannelsConfig{
			Telegram: config.TelegramConfig{
				AllowDirect: true,
			},
		},
	}
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", cfg.Channels.Telegram, messageBus, nil),
		config:      cfg,
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "/new",
		MessageID: 9,
		Chat: telego.Chat{
			ID:   123,
			Type: "private",
		},
		From: &telego.User{
			ID:        42,
			FirstName: "Alice",
		},
	}

	if err := ch.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be forwarded")
	}
	if inbound.Channel != "telegram" {
		t.Fatalf("channel=%q", inbound.Channel)
	}
	if inbound.Content != "/new" {
		t.Fatalf("content=%q", inbound.Content)
	}
}
