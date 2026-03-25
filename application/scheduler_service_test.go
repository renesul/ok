package application

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestSchedulerService_CreateJob_Valid(t *testing.T) {
	repo := &mockJobRepo{}
	svc := NewSchedulerService(repo, zap.NewNop())

	job, err := svc.CreateJob(context.Background(), "build check", "agent", "verificar build", 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID == "" {
		t.Error("expected UUID to be set")
	}
	if !job.Enabled {
		t.Error("expected job to be enabled")
	}
	if job.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if !repo.createCalled {
		t.Error("expected repo.Create to be called")
	}
}

func TestSchedulerService_CreateJob_MissingName(t *testing.T) {
	svc := NewSchedulerService(&mockJobRepo{}, zap.NewNop())
	_, err := svc.CreateJob(context.Background(), "", "agent", "input", 60)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSchedulerService_CreateJob_InvalidInterval(t *testing.T) {
	svc := NewSchedulerService(&mockJobRepo{}, zap.NewNop())
	_, err := svc.CreateJob(context.Background(), "job", "agent", "input", 59)
	if err == nil {
		t.Fatal("expected error for interval < 60")
	}
}

func TestSchedulerService_CreateJob_MinInterval(t *testing.T) {
	svc := NewSchedulerService(&mockJobRepo{}, zap.NewNop())
	job, err := svc.CreateJob(context.Background(), "job", "agent", "input", 60)
	if err != nil {
		t.Fatalf("60s should be accepted: %v", err)
	}
	if job.IntervalSeconds != 60 {
		t.Errorf("interval = %d, want 60", job.IntervalSeconds)
	}
}
