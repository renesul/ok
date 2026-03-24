package integration

import (
	"encoding/json"
	"strings"
	"testing"

	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	sched "github.com/renesul/ok/infrastructure/scheduler"
	"go.uber.org/zap"
)

func TestScheduleTaskBasic(t *testing.T) {
	defer cleanupJobs(t)

	jobRepo := sched.NewJobRepository(testDB, zap.NewNop())
	tool := agenttools.NewScheduleTaskTool(jobRepo)

	input, _ := json.Marshal(map[string]interface{}{
		"name":             "verificar build",
		"input":            "verificar se o build esta passando",
		"interval_minutes": 30,
	})

	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "verificar build") {
		t.Errorf("expected result to contain job name, got: %s", result)
	}
	if !strings.Contains(result, "30 minutos") {
		t.Errorf("expected result to contain interval, got: %s", result)
	}

	// Verify job exists in database
	jobs, err := jobRepo.FindAll(nil)
	if err != nil {
		t.Fatalf("find jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Name != "verificar build" {
		t.Errorf("expected job name 'verificar build', got '%s'", jobs[0].Name)
	}
	if jobs[0].IntervalSeconds != 1800 {
		t.Errorf("expected interval 1800s, got %d", jobs[0].IntervalSeconds)
	}
	if jobs[0].TaskType != "agent" {
		t.Errorf("expected task_type 'agent', got '%s'", jobs[0].TaskType)
	}
	if !jobs[0].Enabled {
		t.Error("expected job to be enabled")
	}
}

func TestScheduleTaskMissingName(t *testing.T) {
	jobRepo := sched.NewJobRepository(testDB, zap.NewNop())
	tool := agenttools.NewScheduleTaskTool(jobRepo)

	input, _ := json.Marshal(map[string]interface{}{
		"input": "alguma coisa",
	})

	_, err := tool.Run(string(input))
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestScheduleTaskDefaultInterval(t *testing.T) {
	defer cleanupJobs(t)

	jobRepo := sched.NewJobRepository(testDB, zap.NewNop())
	tool := agenttools.NewScheduleTaskTool(jobRepo)

	input, _ := json.Marshal(map[string]interface{}{
		"name":  "lembrete",
		"input": "verificar emails",
	})

	_, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs, err := jobRepo.FindAll(nil)
	if err != nil {
		t.Fatalf("find jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	// Default interval: 60 minutes = 3600 seconds
	if jobs[0].IntervalSeconds != 3600 {
		t.Errorf("expected default interval 3600s, got %d", jobs[0].IntervalSeconds)
	}
}

