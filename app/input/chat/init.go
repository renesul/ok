package chat

import (
	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func init() {
	channels.RegisterFactory("chat", func(cfg *config.Config, b *events.MessageBus) (channels.Channel, error) {
		return NewChatChannel(cfg, b), nil
	})
}
