package application

import (
	"context"
	"testing"

	"github.com/renesul/ok/domain"
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

func TestSchedulerService_UpdateJob_Enable(t *testing.T) {
	enabled := true
	repo := &mockJobRepo{
		findByIDResult: &domain.Job{ID: "j1", FailCount: 5, Enabled: false},
	}
	svc := NewSchedulerService(repo, zap.NewNop())

	job, err := svc.UpdateJob(context.Background(), "j1", &enabled, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !job.Enabled {
		t.Error("expected job to be enabled")
	}
	if job.FailCount != 0 {
		t.Errorf("FailCount should reset to 0 on enable, got %d", job.FailCount)
	}
	if !repo.updateCalled {
		t.Error("expected repo.Update to be called")
	}
}

func TestSchedulerService_UpdateJob_InvalidInterval(t *testing.T) {
	interval := 30
	repo := &mockJobRepo{
		findByIDResult: &domain.Job{ID: "j1"},
	}
	svc := NewSchedulerService(repo, zap.NewNop())

	_, err := svc.UpdateJob(context.Background(), "j1", nil, &interval)
	if err == nil {
		t.Fatal("expected error for interval < 60")
	}
}

func TestSchedulerService_UpdateJob_NotFound(t *testing.T) {
	enabled := true
	repo := &mockJobRepo{findByIDResult: nil}
	svc := NewSchedulerService(repo, zap.NewNop())

	_, err := svc.UpdateJob(context.Background(), "nonexistent", &enabled, nil)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestSchedulerService_DeleteJob(t *testing.T) {
	repo := &mockJobRepo{}
	svc := NewSchedulerService(repo, zap.NewNop())

	err := svc.DeleteJob(context.Background(), "j1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.deleteCalled {
		t.Error("expected repo.Delete to be called")
	}
}
