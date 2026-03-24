package adapters

import (
	"context"
	"fmt"
	"os"

	"github.com/renesul/ok/application"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"

	_ "github.com/glebarez/sqlite"
)

type WhatsAppAdapter struct {
	agentService *application.AgentService
	ownerNumber  string
	dbPath       string
	client       *whatsmeow.Client
	log          *zap.Logger
}

func NewWhatsAppAdapter(agentService *application.AgentService, ownerNumber, dbPath string, log *zap.Logger) *WhatsAppAdapter {
	return &WhatsAppAdapter{
		agentService: agentService,
		ownerNumber:  ownerNumber,
		dbPath:       dbPath,
		log:          log.Named("adapter.whatsapp"),
	}
}

func (a *WhatsAppAdapter) Enabled() bool {
	return a.ownerNumber != "" && a.dbPath != ""
}

func (a *WhatsAppAdapter) Start() {
	if !a.Enabled() {
		a.log.Debug("whatsapp adapter disabled")
		return
	}

	a.log.Debug("whatsapp starting", zap.String("owner", a.ownerNumber))

	container, err := sqlstore.New(context.Background(), "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", a.dbPath), waLog.Noop)
	if err != nil {
		a.log.Error("whatsapp store error", zap.Error(err))
		return
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		a.log.Error("whatsapp device error", zap.Error(err))
		return
	}

	a.client = whatsmeow.NewClient(deviceStore, waLog.Noop)
	a.client.AddEventHandler(a.eventHandler)

	if a.client.Store.ID == nil {
		qrChan, _ := a.client.GetQRChannel(context.Background())
		err = a.client.Connect()
		if err != nil {
			a.log.Error("whatsapp connect error", zap.Error(err))
			return
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Fprintf(os.Stderr, "\n[WhatsApp] Escaneie o QR Code:\n%s\n\n", evt.Code)
			}
		}
	} else {
		err = a.client.Connect()
		if err != nil {
			a.log.Error("whatsapp connect error", zap.Error(err))
			return
		}
	}

	a.log.Debug("whatsapp connected")
}

func (a *WhatsAppAdapter) Stop() {
	if a.client != nil {
		a.client.Disconnect()
		a.log.Debug("whatsapp disconnected")
	}
}

func (a *WhatsAppAdapter) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		a.handleMessage(v)
	}
}

func (a *WhatsAppAdapter) handleMessage(msg *events.Message) {
	// Ignorar grupos
	if msg.Info.IsGroup {
		return
	}

	sender := msg.Info.Sender.User

	// Ignorar qualquer numero que nao seja o owner
	if sender != a.ownerNumber {
		a.log.Debug("whatsapp ignored", zap.String("from", sender))
		return
	}

	text := extractText(msg.Message)
	if text == "" {
		return
	}

	a.log.Debug("whatsapp owner command", zap.String("text", text))

	resp, err := a.agentService.Run(context.Background(), text)
	if err != nil {
		a.log.Debug("whatsapp agent error", zap.Error(err))
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("whatsapp agent result", zap.String("result", result))

	// NAO envia resposta — modo passivo
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}
	return ""
}
