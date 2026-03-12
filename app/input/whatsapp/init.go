package whatsapp

import (
	"path/filepath"

	events "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
)

func init() {
	channels.RegisterFactory("whatsapp", func(cfg *config.Config, b *events.MessageBus) (channels.Channel, error) {
		waCfg := cfg.Channels.WhatsApp
		storePath := waCfg.SessionStorePath
		if storePath == "" {
			storePath = filepath.Join(cfg.WorkspacePath(), "whatsapp")
		}
		return NewWhatsAppChannel(waCfg, b, storePath)
	})
}
