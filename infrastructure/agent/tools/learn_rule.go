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
func (t *LearnRuleTool) Description() string                { return "records a permanent rule that the agent must always obey (JSON: {\"rule\":\"...\"})" }
func (t *LearnRuleTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *LearnRuleTool) Run(input string) (string, error) {
	var req struct {
		Rule string `json:"rule"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input must be JSON: {\"rule\":\"prefer Fiber over Echo\"}")
	}

	if req.Rule == "" {
		return "", fmt.Errorf("rule required")
	}

	if err := t.memory.Save(domain.MemoryEntry{
		Content:  req.Rule,
		Category: "rule",
	}); err != nil {
		return "", fmt.Errorf("save rule: %w", err)
	}

	return fmt.Sprintf("rule learned: %s", req.Rule), nil
}
