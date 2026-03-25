package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func newTestPlanner(tools ...domain.Tool) *DefaultPlanner {
	p := NewDefaultPlanner(zap.NewNop())
	for _, t := range tools {
		p.RegisterTool(t)
	}
	return p
}

func TestPlanner_DecisionDone_ReturnsErrDone(t *testing.T) {
	p := newTestPlanner()
	ctx := domain.NewAgentContext(10)
	_, err := p.Plan(domain.Decision{Done: true}, ctx)
	if !errors.Is(err, ErrDone) {
		t.Fatalf("expected ErrDone, got %v", err)
	}
}

func TestPlanner_LimitReached(t *testing.T) {
	tool := &testTool{name: "echo", safety: domain.ToolSafe}
	p := newTestPlanner(tool)
	ctx := domain.NewAgentContext(3)
	ctx.Steps = 3 // already at limit
	_, err := p.Plan(domain.Decision{Tool: "echo", Input: "hi"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "limite") {
		t.Fatalf("expected limite error, got %v", err)
	}
}

func TestPlanner_UnknownTool(t *testing.T) {
	p := newTestPlanner()
	ctx := domain.NewAgentContext(10)
	_, err := p.Plan(domain.Decision{Tool: "nonexistent", Input: "x"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "nao encontrada") {
		t.Fatalf("expected 'nao encontrada' error, got %v", err)
	}
}

func TestPlanner_EmptyInput(t *testing.T) {
	tool := &testTool{name: "shell", safety: domain.ToolDangerous}
	p := newTestPlanner(tool)
	ctx := domain.NewAgentContext(10)
	_, err := p.Plan(domain.Decision{Tool: "shell", Input: ""}, ctx)
	if err == nil || !strings.Contains(err.Error(), "input vazio") {
		t.Fatalf("expected 'input vazio' error, got %v", err)
	}
}

func TestPlanner_EmptyInput_EchoAllowed(t *testing.T) {
	tool := &testTool{name: "echo", safety: domain.ToolSafe}
	p := newTestPlanner(tool)
	ctx := domain.NewAgentContext(10)
	plan, err := p.Plan(domain.Decision{Tool: "echo", Input: ""}, ctx)
	if err != nil {
		t.Fatalf("echo with empty input should succeed: %v", err)
	}
	if plan.Tool.Name() != "echo" {
		t.Fatalf("expected tool echo, got %s", plan.Tool.Name())
	}
}

func TestPlanner_ValidDecision(t *testing.T) {
	tool := &testTool{name: "echo", safety: domain.ToolSafe}
	p := newTestPlanner(tool)
	ctx := domain.NewAgentContext(10)
	plan, err := p.Plan(domain.Decision{Tool: "echo", Input: "hello"}, ctx)
	if err != nil {
		t.Fatalf("valid decision should succeed: %v", err)
	}
	if plan.Tool.Name() != "echo" {
		t.Fatalf("expected tool echo, got %s", plan.Tool.Name())
	}
	if plan.Input != "hello" {
		t.Fatalf("expected input 'hello', got %q", plan.Input)
	}
}

func TestPlanner_RegisterTool_AppearsInTools(t *testing.T) {
	t1 := &testTool{name: "echo", safety: domain.ToolSafe}
	t2 := &testTool{name: "shell", safety: domain.ToolDangerous}
	p := newTestPlanner(t1, t2)
	tools := p.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if _, ok := tools["echo"]; !ok {
		t.Fatal("echo not found in tools")
	}
	if _, ok := tools["shell"]; !ok {
		t.Fatal("shell not found in tools")
	}
}

func TestPlanner_ToolDescriptions_ContainsSafety(t *testing.T) {
	tool := &testTool{name: "shell", safety: domain.ToolDangerous}
	p := newTestPlanner(tool)
	desc := p.ToolDescriptions()
	if !strings.Contains(desc, "[dangerous]") {
		t.Fatalf("expected '[dangerous]' in descriptions, got %q", desc)
	}
	if !strings.Contains(desc, "shell") {
		t.Fatalf("expected 'shell' in descriptions, got %q", desc)
	}
}

func TestPlanner_EmptyToolName(t *testing.T) {
	p := newTestPlanner()
	ctx := domain.NewAgentContext(10)
	_, err := p.Plan(domain.Decision{Tool: "", Input: "x"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "nao encontrada") {
		t.Fatalf("expected 'nao encontrada' error for empty tool name, got %v", err)
	}
}
