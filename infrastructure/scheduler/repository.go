package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type JobRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewJobRepository(db *sql.DB, log *zap.Logger) *JobRepository {
	return &JobRepository{db: db, log: log.Named("repository.job")}
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("insert job", zap.String("name", job.Name))
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO scheduled_jobs (id, name, task_type, input, interval_seconds, enabled, last_run, last_status, fail_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		job.ID, job.Name, job.TaskType, job.Input, job.IntervalSeconds, job.Enabled, job.LastRun, job.LastStatus, job.FailCount, job.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (r *JobRepository) FindAll(ctx context.Context) ([]domain.Job, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find all jobs")
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, task_type, input, interval_seconds, enabled, last_run, last_status, fail_count, created_at FROM scheduled_jobs ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("find all jobs: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (r *JobRepository) FindByID(ctx context.Context, id string) (*domain.Job, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find job by id", zap.String("id", id))
	var job domain.Job
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, task_type, input, interval_seconds, enabled, last_run, last_status, fail_count, created_at FROM scheduled_jobs WHERE id = ?", id,
	).Scan(&job.ID, &job.Name, &job.TaskType, &job.Input, &job.IntervalSeconds, &job.Enabled, &job.LastRun, &job.LastStatus, &job.FailCount, &job.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find job: %w", err)
	}
	return &job, nil
}

func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("update job", zap.String("id", job.ID))
	_, err := r.db.ExecContext(ctx,
		"UPDATE scheduled_jobs SET name=?, task_type=?, input=?, interval_seconds=?, enabled=?, last_run=?, last_status=?, fail_count=? WHERE id=?",
		job.Name, job.TaskType, job.Input, job.IntervalSeconds, job.Enabled, job.LastRun, job.LastStatus, job.FailCount, job.ID,
	)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return nil
}

func (r *JobRepository) Delete(ctx context.Context, id string) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete job", zap.String("id", id))
	_, err := r.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}

func (r *JobRepository) FindEnabled(ctx context.Context) ([]domain.Job, error) {
	ctx = database.Ctx(ctx)
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, task_type, input, interval_seconds, enabled, last_run, last_status, fail_count, created_at FROM scheduled_jobs WHERE enabled = 1")
	if err != nil {
		return nil, fmt.Errorf("find enabled jobs: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (r *JobRepository) UpdateRun(ctx context.Context, id string, status string, failCount int, enabled bool) error {
	ctx = database.Ctx(ctx)
	_, err := r.db.ExecContext(ctx,
		"UPDATE scheduled_jobs SET last_run=?, last_status=?, fail_count=?, enabled=? WHERE id=?",
		time.Now(), status, failCount, enabled, id,
	)
	if err != nil {
		return fmt.Errorf("update job run: %w", err)
	}
	return nil
}

func scanJobs(rows *sql.Rows) ([]domain.Job, error) {
	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		if err := rows.Scan(&job.ID, &job.Name, &job.TaskType, &job.Input, &job.IntervalSeconds, &job.Enabled, &job.LastRun, &job.LastStatus, &job.FailCount, &job.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}
