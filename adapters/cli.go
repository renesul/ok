package adapters

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

type CLIAdapter struct {
	agentRunner AgentRunner
	log         *zap.Logger
}

func NewCLIAdapter(agentRunner AgentRunner, log *zap.Logger) *CLIAdapter {
	return &CLIAdapter{
		agentRunner: agentRunner,
		log:          log.Named("adapter.cli"),
	}
}

func (a *CLIAdapter) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("OK Agent CLI — digite 'exit' para sair")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Ate mais.")
			break
		}

		a.log.Debug("cli input", zap.String("input", input))

		resp, err := a.agentRunner.Run(context.Background(), input)
		if err != nil {
			fmt.Printf("Erro: %s\n\n", err.Error())
			continue
		}

		output := NormalizeResponse(resp)
		fmt.Println(output)

		if len(resp.Memory) > 0 {
			fmt.Printf("\nMemoria: %s\n", strings.Join(resp.Memory, " | "))
		}

		fmt.Println()
	}
}
