package adapters

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

type DiscordAdapter struct {
	agentService *application.AgentService
	botToken     string
	ownerID      string
	session      *discordgo.Session
	log          *zap.Logger
}

func NewDiscordAdapter(agentService *application.AgentService, botToken, ownerID string, log *zap.Logger) *DiscordAdapter {
	return &DiscordAdapter{
		agentService: agentService,
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

	resp, err := a.agentService.Run(context.Background(), text)
	if err != nil {
		a.log.Debug("discord agent error", zap.Error(err))
		return
	}

	result := NormalizeResponse(resp)
	a.log.Debug("discord agent result", zap.String("result", result))

	// NAO envia resposta — modo passivo
}
