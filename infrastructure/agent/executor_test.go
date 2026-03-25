package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type execTestTool struct {
	name   string
	result string
	err    error
}

func (t *execTestTool) Name() string                        { return t.name }
func (t *execTestTool) Description() string                 { return "test" }
func (t *execTestTool) Run(input string) (string, error)    { return t.result, t.err }
func (t *execTestTool) Safety() domain.ToolSafety           { return domain.ToolSafe }

func TestExecutor_Success(t *testing.T) {
	executor := NewDefaultExecutor(zap.NewNop())
	tool := &execTestTool{name: "echo", result: "hello"}

	result, err := executor.Execute(domain.Plan{Tool: tool, Input: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("result = %q, want 'hello'", result)
	}
}

func TestExecutor_ToolError(t *testing.T) {
	executor := NewDefaultExecutor(zap.NewNop())
	tool := &execTestTool{name: "fail", err: fmt.Errorf("tool error")}

	_, err := executor.Execute(domain.Plan{Tool: tool, Input: "x"})
	if err == nil {
		t.Fatal("expected error from tool")
	}
	if !strings.Contains(err.Error(), "falhou") {
		t.Errorf("error = %q, want to contain 'falhou'", err.Error())
	}
}

func TestExecutor_RateLimited(t *testing.T) {
	executor := NewDefaultExecutor(zap.NewNop())
	tool := &execTestTool{name: "shell", result: "ok"}

	// Exhaust rate limit (5/min for shell)
	for i := 0; i < 5; i++ {
		executor.Execute(domain.Plan{Tool: tool, Input: "echo"})
	}

	_, err := executor.Execute(domain.Plan{Tool: tool, Input: "echo"})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %q, want 'rate limit'", err.Error())
	}
}

func TestExecutor_SafetyBlocked(t *testing.T) {
	executor := NewDefaultExecutor(zap.NewNop())
	dangerousTool := &testTool{name: "shell", safety: domain.ToolDangerous}

	_, err := executor.Execute(domain.Plan{Tool: dangerousTool, Input: "rm -rf"})
	if err == nil {
		t.Fatal("expected safety error")
	}
	if !strings.Contains(err.Error(), "confirmation") {
		t.Errorf("error = %q, want 'confirmation'", err.Error())
	}
}
