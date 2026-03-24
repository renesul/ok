package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

const minIntervalSeconds = 60

type SchedulerService struct {
	repository domain.JobRepository
	log        *zap.Logger
}

func NewSchedulerService(repository domain.JobRepository, log *zap.Logger) *SchedulerService {
	return &SchedulerService{
		repository: repository,
		log:        log.Named("service.scheduler"),
	}
}

func (s *SchedulerService) CreateJob(ctx context.Context, name, taskType, input string, intervalSeconds int) (*domain.Job, error) {
	if name == "" {
		return nil, fmt.Errorf("name obrigatorio")
	}
	if taskType == "" {
		return nil, fmt.Errorf("task_type obrigatorio")
	}
	if input == "" {
		return nil, fmt.Errorf("input obrigatorio")
	}
	if intervalSeconds < minIntervalSeconds {
		return nil, fmt.Errorf("intervalo minimo: %d segundos", minIntervalSeconds)
	}

	job := &domain.Job{
		ID:              uuid.New().String(),
		Name:            name,
		TaskType:        taskType,
		Input:           input,
		IntervalSeconds: intervalSeconds,
		Enabled:         true,
		CreatedAt:       time.Now(),
	}

	if err := s.repository.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	s.log.Debug("job created", zap.String("id", job.ID), zap.String("name", name))
	return job, nil
}

func (s *SchedulerService) ListJobs(ctx context.Context) ([]domain.Job, error) {
	return s.repository.FindAll(ctx)
}

func (s *SchedulerService) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *SchedulerService) UpdateJob(ctx context.Context, id string, enabled *bool, intervalSeconds *int) (*domain.Job, error) {
	job, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find job: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("job nao encontrado")
	}

	if enabled != nil {
		job.Enabled = *enabled
		if *enabled {
			job.FailCount = 0
		}
	}
	if intervalSeconds != nil {
		if *intervalSeconds < minIntervalSeconds {
			return nil, fmt.Errorf("intervalo minimo: %d segundos", minIntervalSeconds)
		}
		job.IntervalSeconds = *intervalSeconds
	}

	if err := s.repository.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}

	return job, nil
}

func (s *SchedulerService) DeleteJob(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}
