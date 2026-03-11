package discord

import (
	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/channels"
	"github.com/renesul/ok/pkg/config"
)

func init() {
	channels.RegisterFactory("discord", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewDiscordChannel(cfg.Channels.Discord, b)
	})
}
