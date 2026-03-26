package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/renesul/ok/domain"
)

const maxSubAgents = 3

// SubEngineRunner executa um sub-agente e retorna as mensagens
type SubEngineRunner func(ctx context.Context, input string) ([]string, error)

type DelegateTaskTool struct {
	runner    SubEngineRunner
	callCount atomic.Int32
}

func NewDelegateTaskTool(runner SubEngineRunner) *DelegateTaskTool {
	return &DelegateTaskTool{runner: runner}
}

func (t *DelegateTaskTool) Name() string { return "delegate" }
func (t *DelegateTaskTool) Description() string {
	return "Delegates a sub-task to an independent sub-agent with a clean context window. Input JSON: {\"sub_task\":\"description\", \"context\":\"additional context\"}. Use for complex tasks that benefit from isolated focus. Max 3 delegations per execution."
}
func (t *DelegateTaskTool) Safety() domain.ToolSafety { return domain.ToolRestricted }

func (t *DelegateTaskTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *DelegateTaskTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req struct {
		SubTask string `json:"sub_task"`
		Context string `json:"context"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input must be JSON: {\"sub_task\":\"description\", \"context\":\"info\"}")
	}

	if req.SubTask == "" {
		return "", fmt.Errorf("sub_task required")
	}

	// Limite de sub-agentes por execucao
	count := t.callCount.Add(1)
	if count > maxSubAgents {
		return "", fmt.Errorf("limit of %d sub-agents reached", maxSubAgents)
	}

	// Montar input para o sub-agente
	subInput := req.SubTask
	if req.Context != "" {
		subInput = req.Context + "\n\nTask: " + req.SubTask
	}

	messages, err := t.runner(ctx, subInput)
	if err != nil {
		return "", fmt.Errorf("sub-agent failed: %w", err)
	}

	if len(messages) == 0 {
		return "sub-agent finished without output", nil
	}

	return strings.Join(messages, "\n"), nil
}

// ResetCount reseta o contador de delegacoes (para nova execucao do master)
func (t *DelegateTaskTool) ResetCount() {
	t.callCount.Store(0)
}
