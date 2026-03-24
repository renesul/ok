package tools

import (
	"encoding/json"
	"fmt"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
)

type LearnRuleTool struct {
	memory *agent.SQLiteMemory
}

func NewLearnRuleTool(memory *agent.SQLiteMemory) *LearnRuleTool {
	return &LearnRuleTool{memory: memory}
}

func (t *LearnRuleTool) Name() string                       { return "learn_rule" }
func (t *LearnRuleTool) Description() string                { return "grava uma regra permanente que o agente sempre obedecera (JSON: {\"rule\":\"...\"})" }
func (t *LearnRuleTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *LearnRuleTool) Run(input string) (string, error) {
	var req struct {
		Rule string `json:"rule"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"rule\":\"preferir Fiber a Echo\"}")
	}

	if req.Rule == "" {
		return "", fmt.Errorf("rule obrigatorio")
	}

	if err := t.memory.Save(domain.MemoryEntry{
		Content:  req.Rule,
		Category: "rule",
	}); err != nil {
		return "", fmt.Errorf("salvar regra: %w", err)
	}

	return fmt.Sprintf("regra aprendida: %s", req.Rule), nil
}
