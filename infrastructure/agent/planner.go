package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

var ErrDone = errors.New("agent done")

type DefaultPlanner struct {
	tools map[string]domain.Tool
	log   *zap.Logger
}

func NewDefaultPlanner(log *zap.Logger) *DefaultPlanner {
	return &DefaultPlanner{
		tools: make(map[string]domain.Tool),
		log:   log.Named("agent.planner"),
	}
}

func (p *DefaultPlanner) RegisterTool(tool domain.Tool) {
	p.tools[tool.Name()] = tool
	p.log.Debug("tool registered", zap.String("name", tool.Name()))
}

func (p *DefaultPlanner) Plan(decision domain.Decision, ctx *domain.AgentContext) (domain.Plan, error) {
	if decision.Done {
		return domain.Plan{}, ErrDone
	}

	if ctx.LimitReached() {
		p.log.Debug("planner rejected: limit reached", zap.Int("steps", ctx.Steps), zap.Int("max", ctx.MaxSteps))
		return domain.Plan{}, fmt.Errorf("limite de %d steps atingido", ctx.MaxSteps)
	}

	tool, exists := p.tools[decision.Tool]
	if !exists {
		p.log.Debug("planner rejected: tool not found", zap.String("tool", decision.Tool))
		return domain.Plan{}, fmt.Errorf("tool '%s' nao encontrada", decision.Tool)
	}

	if decision.Input == "" && decision.Tool != "echo" {
		p.log.Debug("planner rejected: empty input", zap.String("tool", decision.Tool))
		return domain.Plan{}, fmt.Errorf("input vazio para tool '%s'", decision.Tool)
	}

	p.log.Debug("plan approved", zap.String("tool", decision.Tool), zap.String("input", decision.Input))
	return domain.Plan{Tool: tool, Input: decision.Input}, nil
}

func (p *DefaultPlanner) Tools() map[string]domain.Tool {
	return p.tools
}

func (p *DefaultPlanner) ToolDescriptions() string {
	var descs []string
	for _, tool := range p.tools {
		line := fmt.Sprintf("- %s: %s", tool.Name(), tool.Description())
		if st, ok := tool.(domain.SafeTool); ok {
			line += fmt.Sprintf(" [%s]", st.Safety())
		}
		descs = append(descs, line)
	}
	return strings.Join(descs, "\n")
}
