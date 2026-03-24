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
	return "Delega uma sub-tarefa para um sub-agente independente com janela de contexto limpa. Input JSON: {\"sub_task\":\"descricao\", \"context\":\"contexto adicional\"}. Use para tarefas complexas que se beneficiam de foco isolado. Max 3 delegacoes por execucao."
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
		return "", fmt.Errorf("input deve ser JSON: {\"sub_task\":\"descricao\", \"context\":\"info\"}")
	}

	if req.SubTask == "" {
		return "", fmt.Errorf("sub_task obrigatorio")
	}

	// Limite de sub-agentes por execucao
	count := t.callCount.Add(1)
	if count > maxSubAgents {
		return "", fmt.Errorf("limite de %d sub-agentes atingido", maxSubAgents)
	}

	// Montar input para o sub-agente
	subInput := req.SubTask
	if req.Context != "" {
		subInput = req.Context + "\n\nTarefa: " + req.SubTask
	}

	messages, err := t.runner(ctx, subInput)
	if err != nil {
		return "", fmt.Errorf("sub-agente falhou: %w", err)
	}

	if len(messages) == 0 {
		return "sub-agente concluiu sem output", nil
	}

	return strings.Join(messages, "\n"), nil
}

// ResetCount reseta o contador de delegacoes (para nova execucao do master)
func (t *DelegateTaskTool) ResetCount() {
	t.callCount.Store(0)
}
