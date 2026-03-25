package adapters

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type TelegramAdapter struct {
	agentRunner AgentRunner
	botToken    string
	ownerID     int64
	bot         *tgbotapi.BotAPI
	log         *zap.Logger
}

func NewTelegramAdapter(agentRunner AgentRunner, botToken string, ownerID int64, log *zap.Logger) *TelegramAdapter {
	return &TelegramAdapter{
		agentRunner: agentRunner,
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

	// Semaphore para limitar flood de goroutines (max 50)
	sem := make(chan struct{}, 50)

	for update := range updates {
		if update.Message != nil {
			sem <- struct{}{}
			go func(msg *tgbotapi.Message) {
				defer func() { <-sem }()
				a.handleMessage(msg)
			}(update.Message)
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
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("telegram panic recovered", zap.Any("panic", r))
		}
	}()

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	resp, err := a.agentRunner.Run(ctx, text)
	if err != nil {
		a.log.Debug("telegram agent error", zap.Error(err))
		if a.bot != nil {
			a.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Erro interno: "+err.Error()))
		}
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("telegram agent result", zap.String("result", result))
	if a.bot != nil && result != "" {
		a.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, result))
	}
}
