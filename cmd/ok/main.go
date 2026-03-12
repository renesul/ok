// OK - Lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 OK contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ok/cmd/ok/internal"
	"ok/cmd/ok/internal/agent"
	"ok/cmd/ok/internal/auth"
	"ok/cmd/ok/internal/cron"
	"ok/cmd/ok/internal/gateway"
	"ok/cmd/ok/internal/migrate"
	"ok/cmd/ok/internal/onboard"
	"ok/cmd/ok/internal/skills"
	"ok/cmd/ok/internal/status"
	"ok/cmd/ok/internal/version"
)

func NewOKCommand() *cobra.Command {
	short := fmt.Sprintf("%s ok - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "ok",
		Short:   short,
		Example: "ok list",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorGreen = "\033[1;38;2;16;185;129m"
	banner     = "\r\n" +
		colorGreen + " ██████╗ ██╗  ██╗\n" +
		colorGreen + "██╔═══██╗██║ ██╔╝\n" +
		colorGreen + "██║   ██║█████╔╝ \n" +
		colorGreen + "██║   ██║██╔═██╗ \n" +
		colorGreen + "╚██████╔╝██║  ██╗\n" +
		colorGreen + " ╚═════╝ ╚═╝  ╚═╝\n" +
		"\033[0m\r\n"
)

func main() {
	fmt.Printf("%s", banner)
	cmd := NewOKCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
