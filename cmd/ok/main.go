package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/renesul/ok/application"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/repository"
	"github.com/renesul/ok/interfaces/http"
	"github.com/renesul/ok/interfaces/http/handler"
	"github.com/renesul/ok/internal/config"
	"github.com/renesul/ok/internal/logger"
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

	log, err := logger.New(cfg.LogLevel, cfg.Debug)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer log.Sync()

	log.Debug("config loaded",
		zap.String("port", cfg.ServerPort),
		zap.String("database", cfg.DatabasePath),
		zap.String("log_level", cfg.LogLevel),
		zap.Bool("debug", cfg.Debug),
	)

	db, err := database.New(cfg.DatabasePath, cfg.Debug)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	userRepository := repository.NewUserRepository(db, log)
	userService := application.NewUserService(userRepository, log)
	userHandler := handler.NewUserHandler(userService, log)

	app := http.NewServer(userHandler, log)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Debug("shutting down")
		app.Shutdown()
	}()

	log.Debug("server starting", zap.String("port", cfg.ServerPort), zap.Bool("debug", cfg.Debug))
	return app.Listen(":" + cfg.ServerPort)
}
