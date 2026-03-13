package discord

import (
	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func init() {
	channels.RegisterFactory("discord", func(cfg *config.Config, b *events.MessageBus) (channels.Channel, error) {
		return NewDiscordChannel(cfg.Channels.Discord, cfg.Proxy, b)
	})
}
