package tools

import (
	"fmt"
	"strings"

	"github.com/renesul/ok/domain"
)

type SkillLoaderTool struct {
	repo domain.SkillRepository
}

func NewSkillLoaderTool(repo domain.SkillRepository) *SkillLoaderTool {
	return &SkillLoaderTool{repo: repo}
}

func (t *SkillLoaderTool) Name() string { return "skill_loader" }
func (t *SkillLoaderTool) Description() string {
	return "loads skill instructions into context. Input: skill name as plain string (e.g. 'copywriter'). Returns the full skill content for the agent to follow."
}
func (t *SkillLoaderTool) Safety() domain.ToolSafety { return domain.ToolSafe }

func (t *SkillLoaderTool) Run(input string) (string, error) {
	name := strings.TrimSpace(strings.ToLower(input))
	if name == "" {
		return "", fmt.Errorf("skill name required")
	}

	skill, err := t.repo.Get(name)
	if err != nil {
		return "", fmt.Errorf("load skill: %w", err)
	}

	return skill.Content, nil
}
