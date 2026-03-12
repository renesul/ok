package slack

import (
	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func init() {
	channels.RegisterFactory("slack", func(cfg *config.Config, b *events.MessageBus) (channels.Channel, error) {
		return NewSlackChannel(cfg.Channels.Slack, b)
	})
}
