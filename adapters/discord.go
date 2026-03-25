package adapters

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type DiscordAdapter struct {
	agentRunner AgentRunner
	botToken    string
	ownerID     string
	session     *discordgo.Session
	log         *zap.Logger
}

func NewDiscordAdapter(agentRunner AgentRunner, botToken, ownerID string, log *zap.Logger) *DiscordAdapter {
	return &DiscordAdapter{
		agentRunner: agentRunner,
		botToken:     botToken,
		ownerID:      ownerID,
		log:          log.Named("adapter.discord"),
	}
}

func (a *DiscordAdapter) Enabled() bool {
	return a.botToken != "" && a.ownerID != ""
}

func (a *DiscordAdapter) Start() {
	if !a.Enabled() {
		a.log.Debug("discord adapter disabled")
		return
	}

	var err error
	a.session, err = discordgo.New("Bot " + a.botToken)
	if err != nil {
		a.log.Error("discord session error", zap.Error(err))
		return
	}

	a.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	a.session.AddHandler(a.handleMessage)

	if err := a.session.Open(); err != nil {
		a.log.Error("discord connect error", zap.Error(err))
		return
	}

	a.log.Debug("discord connected", zap.String("bot", a.session.State.User.Username), zap.String("owner_id", a.ownerID))
}

func (a *DiscordAdapter) Stop() {
	if a.session != nil {
		a.session.Close()
		a.log.Debug("discord disconnected")
	}
}

func (a *DiscordAdapter) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("discord panic recovered", zap.Any("panic", r))
		}
	}()

	// Ignorar mensagens do proprio bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignorar qualquer um que nao seja o owner
	if m.Author.ID != a.ownerID {
		a.log.Debug("discord ignored", zap.String("from", m.Author.Username))
		return
	}

	text := m.Content
	if text == "" {
		return
	}

	a.log.Debug("discord owner command", zap.String("text", text))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	resp, err := a.agentRunner.Run(ctx, text)
	if err != nil {
		a.log.Debug("discord agent error", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "⚠️ Erro interno: "+err.Error())
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("discord agent result", zap.String("result", result))
	if result != "" {
		s.ChannelMessageSend(m.ChannelID, result)
	}
}
