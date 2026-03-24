package adapters

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

type TelegramAdapter struct {
	agentService *application.AgentService
	botToken     string
	ownerID      int64
	bot          *tgbotapi.BotAPI
	log          *zap.Logger
}

func NewTelegramAdapter(agentService *application.AgentService, botToken string, ownerID int64, log *zap.Logger) *TelegramAdapter {
	return &TelegramAdapter{
		agentService: agentService,
		botToken:     botToken,
		ownerID:      ownerID,
		log:          log.Named("adapter.telegram"),
	}
}

func (a *TelegramAdapter) Enabled() bool {
	return a.botToken != "" && a.ownerID != 0
}

func (a *TelegramAdapter) Start() {
	if !a.Enabled() {
		a.log.Debug("telegram adapter disabled")
		return
	}

	var err error
	a.bot, err = tgbotapi.NewBotAPI(a.botToken)
	if err != nil {
		a.log.Error("telegram bot error", zap.Error(err))
		return
	}

	a.log.Debug("telegram connected", zap.String("bot", a.bot.Self.UserName), zap.Int64("owner_id", a.ownerID))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := a.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go a.handleMessage(update.Message)
		}
	}
}

func (a *TelegramAdapter) Stop() {
	if a.bot != nil {
		a.bot.StopReceivingUpdates()
		a.log.Debug("telegram disconnected")
	}
}

func (a *TelegramAdapter) handleMessage(msg *tgbotapi.Message) {
	// Ignorar grupos
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		return
	}

	// Ignorar qualquer um que nao seja o owner
	if msg.From == nil || msg.From.ID != a.ownerID {
		a.log.Debug("telegram ignored", zap.Int64("from", msg.From.ID))
		return
	}

	text := msg.Text
	if text == "" {
		return
	}

	a.log.Debug("telegram owner command", zap.String("text", text))

	resp, err := a.agentService.Run(context.Background(), text)
	if err != nil {
		a.log.Debug("telegram agent error", zap.Error(err))
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("telegram agent result", zap.String("result", result))

	// NAO envia resposta — modo passivo
}
