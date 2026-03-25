package engine

import (
	"context"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func newNilHistoryRecorder() *HistoryRecorder {
	return NewHistoryRecorder(nil, nil, nil, zap.NewNop())
}

func TestHistoryRecorder_SearchMemories_NilMemory(t *testing.T) {
	h := newNilHistoryRecorder()
	results, err := h.SearchMemories(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestHistoryRecorder_SaveResults_NilDB(t *testing.T) {
	h := newNilHistoryRecorder()
	state := &domain.ExecutionState{Goal: "test", Phase: domain.PhaseDone}
	// Should not panic
	h.SaveResults(state, "input", "output", time.Now())
}

func TestHistoryRecorder_ReflectAndLearn_NilGuards(t *testing.T) {
	h := newNilHistoryRecorder()
	state := &domain.ExecutionState{Goal: "test"}
	step := &domain.PlannedStep{Name: "s1", Tool: "echo", Status: "done", Output: "ok"}
	reflection := domain.ReflectionResult{Action: "done", Reason: "success"}
	// Should not panic with nil memory and nil db
	h.ReflectAndLearn(context.Background(), state, step, reflection)
}

func TestHistoryRecorder_BuildRecord_Done(t *testing.T) {
	h := newNilHistoryRecorder()
	start := time.Now().Add(-1 * time.Second)
	state := &domain.ExecutionState{
		Goal:  "test goal",
		Phase: domain.PhaseDone,
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Tool: "echo", Status: "done"},
				{Name: "s2", Tool: "shell", Status: "done"},
			},
		},
	}
	record := h.buildExecutionRecord(state, start)
	if record.Status != "done" {
		t.Fatalf("expected status 'done', got %q", record.Status)
	}
	if record.Goal != "test goal" {
		t.Fatalf("expected goal 'test goal', got %q", record.Goal)
	}
	if len(record.ToolsUsed) != 2 {
		t.Fatalf("expected 2 tools used, got %d", len(record.ToolsUsed))
	}
	if record.TotalMs <= 0 {
		t.Fatalf("expected positive TotalMs, got %d", record.TotalMs)
	}
	if record.FailureReason != "" {
		t.Fatalf("expected empty failure reason, got %q", record.FailureReason)
	}
}

func TestHistoryRecorder_BuildRecord_Error(t *testing.T) {
	h := newNilHistoryRecorder()
	state := &domain.ExecutionState{
		Goal:  "test",
		Phase: domain.PhaseError,
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Tool: "shell", Status: "failed"},
			},
		},
	}
	record := h.buildExecutionRecord(state, time.Now())
	if record.Status != "error" {
		t.Fatalf("expected status 'error', got %q", record.Status)
	}
	if record.FailureReason == "" {
		t.Fatal("expected non-empty failure reason for failed step")
	}
}

func TestHistoryRecorder_ExtractResults_SkipsPending(t *testing.T) {
	h := newNilHistoryRecorder()
	state := &domain.ExecutionState{
		Plan: &domain.ExecutionPlan{
			Steps: []domain.PlannedStep{
				{Name: "s1", Tool: "echo", Status: "done"},
				{Name: "s2", Tool: "shell", Status: "pending"},
				{Name: "s3", Tool: "http", Status: "failed"},
			},
		},
	}
	results := h.extractStepResults(state)
	if len(results) != 2 {
		t.Fatalf("expected 2 results (skip pending), got %d", len(results))
	}
	if results[0].Status != "done" || results[1].Status != "failed" {
		t.Fatalf("unexpected results: %v", results)
	}
}
