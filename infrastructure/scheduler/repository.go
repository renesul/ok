package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type JobRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewJobRepository(db *gorm.DB, log *zap.Logger) *JobRepository {
	return &JobRepository{db: db, log: log.Named("repository.job")}
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	r.log.Debug("insert job", zap.String("name", job.Name))
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO scheduled_jobs (id, name, task_type, input, interval_seconds, enabled, last_run, last_status, fail_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		job.ID, job.Name, job.TaskType, job.Input, job.IntervalSeconds, job.Enabled, job.LastRun, job.LastStatus, job.FailCount, job.CreatedAt,
	).Error
}

func (r *JobRepository) FindAll(ctx context.Context) ([]domain.Job, error) {
	r.log.Debug("find all jobs")
	var jobs []domain.Job
	err := r.db.WithContext(ctx).Raw("SELECT * FROM scheduled_jobs ORDER BY created_at DESC").Scan(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("find all jobs: %w", err)
	}
	return jobs, nil
}

func (r *JobRepository) FindByID(ctx context.Context, id string) (*domain.Job, error) {
	r.log.Debug("find job by id", zap.String("id", id))
	var job domain.Job
	err := r.db.WithContext(ctx).Raw("SELECT * FROM scheduled_jobs WHERE id = ?", id).Scan(&job).Error
	if err != nil {
		return nil, fmt.Errorf("find job: %w", err)
	}
	if job.ID == "" {
		return nil, nil
	}
	return &job, nil
}

func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	r.log.Debug("update job", zap.String("id", job.ID))
	return r.db.WithContext(ctx).Exec(
		"UPDATE scheduled_jobs SET name=?, task_type=?, input=?, interval_seconds=?, enabled=?, last_run=?, last_status=?, fail_count=? WHERE id=?",
		job.Name, job.TaskType, job.Input, job.IntervalSeconds, job.Enabled, job.LastRun, job.LastStatus, job.FailCount, job.ID,
	).Error
}

func (r *JobRepository) Delete(ctx context.Context, id string) error {
	r.log.Debug("delete job", zap.String("id", id))
	return r.db.WithContext(ctx).Exec("DELETE FROM scheduled_jobs WHERE id = ?", id).Error
}

func (r *JobRepository) FindEnabled(ctx context.Context) ([]domain.Job, error) {
	var jobs []domain.Job
	err := r.db.WithContext(ctx).Raw("SELECT * FROM scheduled_jobs WHERE enabled = 1").Scan(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("find enabled jobs: %w", err)
	}
	return jobs, nil
}

func (r *JobRepository) UpdateRun(ctx context.Context, id string, status string, failCount int, enabled bool) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE scheduled_jobs SET last_run=?, last_status=?, fail_count=?, enabled=? WHERE id=?",
		time.Now(), status, failCount, enabled, id,
	).Error
}
