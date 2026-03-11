package slack

import (
	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/channels"
	"github.com/renesul/ok/pkg/config"
)

func init() {
	channels.RegisterFactory("slack", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewSlackChannel(cfg.Channels.Slack, b)
	})
}
