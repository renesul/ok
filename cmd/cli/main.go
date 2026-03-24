package main

import (
	"fmt"
	"os"

	"github.com/renesul/ok/adapters"
	"github.com/renesul/ok/infrastructure/bootstrap"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/internal/config"
	"github.com/renesul/ok/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %s\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel, cfg.Debug, cfg.LogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %s\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	db, err := database.New(cfg.DatabasePath, cfg.Debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database: %s\n", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(db); err != nil {
		fmt.Fprintf(os.Stderr, "migrations: %s\n", err)
		os.Exit(1)
	}

	comp := bootstrap.NewAgent(db, cfg, log)

	cli := adapters.NewCLIAdapter(comp.AgentService, log)
	cli.Run()
}
