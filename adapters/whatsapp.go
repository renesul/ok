package adapters

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"

	_ "github.com/glebarez/go-sqlite"
)

type WhatsAppAdapter struct {
	agentRunner AgentRunner
	ownerNumber string
	dbPath      string
	client      *whatsmeow.Client
	log         *zap.Logger
}

func NewWhatsAppAdapter(agentRunner AgentRunner, ownerNumber, dbPath string, log *zap.Logger) *WhatsAppAdapter {
	return &WhatsAppAdapter{
		agentRunner: agentRunner,
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
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("whatsapp panic recovered", zap.Any("panic", r))
		}
	}()

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	resp, err := a.agentRunner.Run(ctx, text)
	if err != nil {
		a.log.Debug("whatsapp agent error", zap.Error(err))
		a.sendText(msg.Info.Chat, "⚠️ Erro interno: "+err.Error())
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("whatsapp agent result", zap.String("result", result))
	if result != "" {
		a.sendText(msg.Info.Chat, result)
	}
}

func (a *WhatsAppAdapter) sendText(to types.JID, text string) {
	if a.client == nil {
		return
	}
	conv := text
	_, err := a.client.SendMessage(context.Background(), to, &waE2E.Message{Conversation: &conv})
	if err != nil {
		a.log.Debug("whatsapp send failed", zap.Error(err))
	}
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
