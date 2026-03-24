package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/renesul/ok/adapters"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/infrastructure/bootstrap"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/llm"
	"github.com/renesul/ok/infrastructure/repository"
	sched "github.com/renesul/ok/infrastructure/scheduler"
	apphttp "github.com/renesul/ok/interfaces/http"
	"github.com/renesul/ok/interfaces/http/handler"
	"github.com/renesul/ok/internal/config"
	"github.com/renesul/ok/internal/logger"

	agent "github.com/renesul/ok/infrastructure/agent"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.LogLevel, cfg.Debug, cfg.LogPath)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer log.Sync()

	log.Debug("config loaded",
		zap.String("port", cfg.ServerPort),
		zap.String("database", cfg.DatabasePath),
		zap.String("log_level", cfg.LogLevel),
		zap.Bool("debug", cfg.Debug),
		zap.Int("auth_password_length", len(cfg.AuthPassword)),
	)

	db, err := database.New(cfg.DatabasePath, cfg.Debug)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	comp := bootstrap.NewAgent(db, cfg, log)

	// Repositories (web-specific)
	sessionRepository := repository.NewSessionRepository(db, log)
	conversationRepository := repository.NewConversationRepository(db, log)
	messageRepository := repository.NewMessageRepository(db, log)

	// Services
	sessionService := application.NewSessionService(sessionRepository, log)
	embeddingService := application.NewEmbeddingService(comp.EmbedClient, messageRepository, conversationRepository, log)
	conversationService := application.NewConversationService(conversationRepository, messageRepository, embeddingService, log)
	importService := application.NewImportService(conversationRepository, messageRepository, embeddingService, log)
	chatService := application.NewChatService(conversationRepository, messageRepository, embeddingService, comp.AgentService, comp.LLMConfigured(), log)

	// Handlers
	authHandler := handler.NewAuthHandler(sessionService, cfg.AuthPassword, log)
	chatHandler := handler.NewChatHandler(conversationService, chatService, log)
	importHandler := handler.NewImportHandler(importService, log)
	healthHandler := handler.NewHealthHandler(comp.LLMClient, llm.ClientConfig{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
	}, comp.EmbedClient, log)
	agentHandler := handler.NewAgentHandler(comp.AgentService, comp.ConfirmManager, handler.ChannelStatus{
		WhatsAppEnabled: cfg.WhatsAppOwnerNumber != "",
		TelegramEnabled: cfg.TelegramBotToken != "" && cfg.TelegramOwnerID != 0,
		DiscordEnabled:  cfg.DiscordBotToken != "" && cfg.DiscordOwnerID != "",
	}, log)

	// Scheduler
	schedulerService := application.NewSchedulerService(comp.JobRepository, log)
	bgScheduler := sched.NewScheduler(comp.JobRepository, comp.AgentService, log)
	go bgScheduler.Start()
	schedulerHandler := handler.NewSchedulerHandler(schedulerService, log)

	// File watcher
	fileWatcher := agent.NewFileWatcher(comp.AgentMemory, cfg.AgentSandboxDir, log)
	go fileWatcher.Start()

	// Memory condenser (a cada 6 horas)
	go func() {
		condenseConfig := comp.LLMFast
		if condenseConfig.BaseURL == "" {
			condenseConfig = comp.LLMHeavy
		}
		for {
			time.Sleep(6 * time.Hour)
			if err := comp.AgentMemory.CondenseOldMemories(context.Background(), comp.LLMClient, condenseConfig); err != nil {
				log.Debug("memory condense failed", zap.Error(err))
			}
		}
	}()

	// Adapters
	whatsappAdapter := adapters.NewWhatsAppAdapter(comp.AgentService, cfg.WhatsAppOwnerNumber, cfg.WhatsAppDBPath, log)
	telegramAdapter := adapters.NewTelegramAdapter(comp.AgentService, cfg.TelegramBotToken, cfg.TelegramOwnerID, log)
	discordAdapter := adapters.NewDiscordAdapter(comp.AgentService, cfg.DiscordBotToken, cfg.DiscordOwnerID, log)

	if whatsappAdapter.Enabled() {
		go whatsappAdapter.Start()
	}
	if telegramAdapter.Enabled() {
		go telegramAdapter.Start()
	}
	if discordAdapter.Enabled() {
		go discordAdapter.Start()
	}

	wsHandler := handler.NewWSHandler(comp.AgentService, comp.ConfirmManager, log)
	app := apphttp.NewServer(authHandler, chatHandler, importHandler, healthHandler, agentHandler, schedulerHandler, wsHandler, sessionService, cfg, log)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Debug("shutting down")
		bgScheduler.Stop()
		whatsappAdapter.Stop()
		telegramAdapter.Stop()
		discordAdapter.Stop()
		app.Shutdown()
	}()

	log.Debug("server starting", zap.String("port", cfg.ServerPort), zap.Bool("debug", cfg.Debug))
	return app.Listen(":" + cfg.ServerPort)
}
