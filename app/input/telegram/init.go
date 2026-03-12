package telegram

import (
	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func init() {
	channels.RegisterFactory("telegram", func(cfg *config.Config, b *events.MessageBus) (channels.Channel, error) {
		return NewTelegramChannel(cfg, b)
	})
}
