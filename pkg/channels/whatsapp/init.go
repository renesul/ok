package whatsapp

import (
	"path/filepath"

	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/channels"
	"github.com/renesul/ok/pkg/config"
)

func init() {
	channels.RegisterFactory("whatsapp", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		waCfg := cfg.Channels.WhatsApp
		storePath := waCfg.SessionStorePath
		if storePath == "" {
			storePath = filepath.Join(cfg.WorkspacePath(), "whatsapp")
		}
		return NewWhatsAppChannel(waCfg, b, storePath)
	})
}
