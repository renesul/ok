package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/renesul/ok/domain"
)

type spyEmitter struct {
	phases   []string
	steps    []string // "name:status"
	messages []string
	done     bool
}

func (s *spyEmitter) EmitPhase(phase string)                         { s.phases = append(s.phases, phase) }
func (s *spyEmitter) EmitStep(name, tool, status string, ms int64)   { s.steps = append(s.steps, name+":"+status) }
func (s *spyEmitter) EmitMessage(content string)                     { s.messages = append(s.messages, content) }
func (s *spyEmitter) EmitDone()                                      { s.done = true }
func (s *spyEmitter) EmitMemories(_ []string)                        {}
func (s *spyEmitter) EmitTerminal(_, _ string)                       {}
func (s *spyEmitter) EmitDiff(_, _, _ string)                        {}
func (s *spyEmitter) EmitConfirm(_, _, _ string)                     {}
func (s *spyEmitter) EmitStream(_, _ string)                         {}

func TestStepRunner_Success(t *testing.T) {
	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("hello result")

	runner := NewStepRunner(planner, executor, nil, nil)
	spy := &spyEmitter{}
	state := &domain.ExecutionState{Goal: "test"}
	decision := domain.Decision{Tool: "echo", Input: "hi"}

	err := runner.ExecuteSingleStep(context.Background(), state, decision, "hi", spy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !spy.done {
		t.Fatal("expected EmitDone called")
	}
	if len(spy.steps) < 1 || spy.steps[len(spy.steps)-1] != "echo:done" {
		t.Fatalf("expected step echo:done, got %v", spy.steps)
	}
	if len(spy.messages) < 1 || spy.messages[len(spy.messages)-1] != "hello result" {
		t.Fatalf("expected message 'hello result', got %v", spy.messages)
	}
}

func TestStepRunner_PlannerRejects(t *testing.T) {
	planner := newMockPlanner()
	planner.planErr = fmt.Errorf("tool rejected")
	executor := newMockExecutor("")

	runner := NewStepRunner(planner, executor, nil, nil)
	spy := &spyEmitter{}
	state := &domain.ExecutionState{Goal: "test"}
	decision := domain.Decision{Tool: "bad", Input: "x"}

	err := runner.ExecuteSingleStep(context.Background(), state, decision, "x", spy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !spy.done {
		t.Fatal("expected EmitDone called")
	}
	found := false
	for _, m := range spy.messages {
		if m == "tool rejected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected rejection message, got %v", spy.messages)
	}
}

func TestStepRunner_ExecutorFails(t *testing.T) {
	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("")
	executor.err = fmt.Errorf("exec failed")

	runner := NewStepRunner(planner, executor, nil, nil)
	spy := &spyEmitter{}
	state := &domain.ExecutionState{Goal: "test"}
	decision := domain.Decision{Tool: "echo", Input: "hi"}

	err := runner.ExecuteSingleStep(context.Background(), state, decision, "hi", spy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !spy.done {
		t.Fatal("expected EmitDone called")
	}
	hasFailStep := false
	for _, s := range spy.steps {
		if s == "echo:failed" {
			hasFailStep = true
		}
	}
	if !hasFailStep {
		t.Fatalf("expected step echo:failed, got %v", spy.steps)
	}
}

func TestLastStepOutput_NilPlan(t *testing.T) {
	state := &domain.ExecutionState{Plan: nil}
	if got := LastStepOutput(state); got != "" {
		t.Fatalf("expected empty string for nil plan, got %q", got)
	}
}

func TestLastStepOutput_PrefersDone(t *testing.T) {
	state := &domain.ExecutionState{
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Status: "failed", Output: "fail output"},
				{Name: "s2", Status: "done", Output: "success output"},
			},
		},
	}
	if got := LastStepOutput(state); got != "success output" {
		t.Fatalf("expected 'success output', got %q", got)
	}
}

func TestLastStepOutput_FallbackFailed(t *testing.T) {
	state := &domain.ExecutionState{
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Status: "failed", Output: "first fail"},
				{Name: "s2", Status: "failed", Output: "last fail"},
			},
		},
	}
	if got := LastStepOutput(state); got != "last fail" {
		t.Fatalf("expected 'last fail', got %q", got)
	}
}

func TestLastStepOutput_EmptyOutputs(t *testing.T) {
	state := &domain.ExecutionState{
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Status: "done", Output: ""},
				{Name: "s2", Status: "failed", Output: ""},
			},
		},
	}
	if got := LastStepOutput(state); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestStepRunner_NilMemory_NoDBPanic(t *testing.T) {
	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("ok")

	runner := NewStepRunner(planner, executor, nil, nil)
	spy := &spyEmitter{}
	state := &domain.ExecutionState{Goal: "test"}

	// Should not panic with nil memory and nil db
	err := runner.ExecuteSingleStep(context.Background(), state, domain.Decision{Tool: "echo", Input: "x"}, "x", spy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !spy.done {
		t.Fatal("expected EmitDone called")
	}
}
