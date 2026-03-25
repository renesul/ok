package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
)

type mockJobRepo struct {
	lastJob *domain.Job
	err     error
}

func (m *mockJobRepo) Create(_ context.Context, job *domain.Job) error {
	m.lastJob = job
	return m.err
}
func (m *mockJobRepo) FindAll(_ context.Context) ([]domain.Job, error)       { return nil, nil }
func (m *mockJobRepo) FindByID(_ context.Context, _ string) (*domain.Job, error) { return nil, nil }
func (m *mockJobRepo) Update(_ context.Context, _ *domain.Job) error         { return nil }
func (m *mockJobRepo) Delete(_ context.Context, _ string) error              { return nil }

func TestSchedule_Success(t *testing.T) {
	repo := &mockJobRepo{}
	tool := NewScheduleTaskTool(repo)
	result, err := tool.Run(`{"name":"backup","input":"run backup","interval_minutes":30}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "agendado") || !strings.Contains(result, "backup") {
		t.Fatalf("expected success message with 'agendado' and 'backup', got %q", result)
	}
	if repo.lastJob == nil {
		t.Fatal("expected job to be persisted")
	}
	if repo.lastJob.IntervalSeconds != 1800 {
		t.Fatalf("expected 1800s interval, got %d", repo.lastJob.IntervalSeconds)
	}
}

func TestSchedule_EmptyName(t *testing.T) {
	tool := NewScheduleTaskTool(&mockJobRepo{})
	_, err := tool.Run(`{"name":"","input":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected 'name' error, got %v", err)
	}
}

func TestSchedule_EmptyInput(t *testing.T) {
	tool := NewScheduleTaskTool(&mockJobRepo{})
	_, err := tool.Run(`{"name":"x","input":""}`)
	if err == nil || !strings.Contains(err.Error(), "input") {
		t.Fatalf("expected 'input' error, got %v", err)
	}
}

func TestSchedule_DefaultInterval(t *testing.T) {
	repo := &mockJobRepo{}
	tool := NewScheduleTaskTool(repo)
	tool.Run(`{"name":"test","input":"do it"}`)
	if repo.lastJob == nil {
		t.Fatal("expected job to be persisted")
	}
	if repo.lastJob.IntervalSeconds != 3600 {
		t.Fatalf("expected default 3600s interval, got %d", repo.lastJob.IntervalSeconds)
	}
}

func TestSchedule_CustomInterval(t *testing.T) {
	repo := &mockJobRepo{}
	tool := NewScheduleTaskTool(repo)
	tool.Run(`{"name":"test","input":"do it","interval_minutes":5}`)
	if repo.lastJob == nil {
		t.Fatal("expected job to be persisted")
	}
	if repo.lastJob.IntervalSeconds != 300 {
		t.Fatalf("expected 300s interval, got %d", repo.lastJob.IntervalSeconds)
	}
}

func TestSchedule_RepoError(t *testing.T) {
	repo := &mockJobRepo{err: fmt.Errorf("db failure")}
	tool := NewScheduleTaskTool(repo)
	_, err := tool.Run(`{"name":"test","input":"do it"}`)
	if err == nil || !strings.Contains(err.Error(), "db failure") {
		t.Fatalf("expected repo error, got %v", err)
	}
}
