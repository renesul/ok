package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
)

const defaultIntervalMinutes = 60

type ScheduleTaskTool struct {
	jobRepo domain.JobRepository
}

func NewScheduleTaskTool(jobRepo domain.JobRepository) *ScheduleTaskTool {
	return &ScheduleTaskTool{jobRepo: jobRepo}
}

func (t *ScheduleTaskTool) Name() string                       { return "schedule" }
func (t *ScheduleTaskTool) Description() string                { return "schedules a task for future periodic execution" }
func (t *ScheduleTaskTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *ScheduleTaskTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *ScheduleTaskTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req struct {
		Name            string `json:"name"`
		Input           string `json:"input"`
		IntervalMinutes int    `json:"interval_minutes"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"name\":\"...\", \"input\":\"...\", \"interval_minutes\":60}")
	}

	if req.Name == "" {
		return "", fmt.Errorf("name obrigatorio")
	}
	if req.Input == "" {
		return "", fmt.Errorf("input obrigatorio")
	}
	if req.IntervalMinutes <= 0 {
		req.IntervalMinutes = defaultIntervalMinutes
	}

	intervalSeconds := req.IntervalMinutes * 60

	job := &domain.Job{
		ID:              uuid.New().String(),
		Name:            req.Name,
		TaskType:        "agent",
		Input:           req.Input,
		IntervalSeconds: intervalSeconds,
		Enabled:         true,
		CreatedAt:       time.Now(),
	}

	if err := t.jobRepo.Create(ctx, job); err != nil {
		return "", fmt.Errorf("criar job: %w", err)
	}

	return fmt.Sprintf("Job agendado: '%s' a cada %d minutos (id: %s)", req.Name, req.IntervalMinutes, job.ID), nil
}
