package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestDelegate_Success(t *testing.T) {
	runner := func(_ context.Context, input string) ([]string, error) {
		return []string{"result1"}, nil
	}
	tool := NewDelegateTaskTool(runner)
	result, err := tool.Run(`{"sub_task":"do something"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result1" {
		t.Fatalf("expected 'result1', got %q", result)
	}
}

func TestDelegate_WithContext(t *testing.T) {
	var capturedInput string
	runner := func(_ context.Context, input string) ([]string, error) {
		capturedInput = input
		return []string{"ok"}, nil
	}
	tool := NewDelegateTaskTool(runner)
	tool.Run(`{"sub_task":"tarefa","context":"background info"}`)
	if !strings.Contains(capturedInput, "background info") {
		t.Fatalf("expected context in input, got %q", capturedInput)
	}
	if !strings.Contains(capturedInput, "tarefa") {
		t.Fatalf("expected sub_task in input, got %q", capturedInput)
	}
}

func TestDelegate_EmptySubTask(t *testing.T) {
	runner := func(_ context.Context, _ string) ([]string, error) { return nil, nil }
	tool := NewDelegateTaskTool(runner)
	_, err := tool.Run(`{"sub_task":""}`)
	if err == nil || !strings.Contains(err.Error(), "obrigatorio") {
		t.Fatalf("expected 'obrigatorio' error, got %v", err)
	}
}

func TestDelegate_MaxSubAgents(t *testing.T) {
	runner := func(_ context.Context, _ string) ([]string, error) { return []string{"ok"}, nil }
	tool := NewDelegateTaskTool(runner)

	for i := 0; i < maxSubAgents; i++ {
		_, err := tool.Run(`{"sub_task":"task"}`)
		if err != nil {
			t.Fatalf("call %d should succeed: %v", i+1, err)
		}
	}

	_, err := tool.Run(`{"sub_task":"task"}`)
	if err == nil || !strings.Contains(err.Error(), "limite") {
		t.Fatalf("expected 'limite' error on call %d, got %v", maxSubAgents+1, err)
	}
}

func TestDelegate_ResetCount(t *testing.T) {
	runner := func(_ context.Context, _ string) ([]string, error) { return []string{"ok"}, nil }
	tool := NewDelegateTaskTool(runner)

	for i := 0; i < maxSubAgents; i++ {
		tool.Run(`{"sub_task":"task"}`)
	}

	tool.ResetCount()
	_, err := tool.Run(`{"sub_task":"task after reset"}`)
	if err != nil {
		t.Fatalf("after ResetCount, call should succeed: %v", err)
	}
}

func TestDelegate_RunnerError(t *testing.T) {
	runner := func(_ context.Context, _ string) ([]string, error) {
		return nil, fmt.Errorf("internal failure")
	}
	tool := NewDelegateTaskTool(runner)
	_, err := tool.Run(`{"sub_task":"task"}`)
	if err == nil || !strings.Contains(err.Error(), "sub-agente falhou") {
		t.Fatalf("expected 'sub-agente falhou' error, got %v", err)
	}
}
