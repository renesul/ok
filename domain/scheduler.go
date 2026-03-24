package domain

import (
	"context"
	"time"
)

type Job struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	TaskType        string    `json:"task_type"`
	Input           string    `json:"input"`
	IntervalSeconds int       `json:"interval_seconds"`
	Enabled         bool      `json:"enabled"`
	LastRun         time.Time `json:"last_run"`
	LastStatus      string    `json:"last_status"`
	FailCount       int       `json:"fail_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	FindAll(ctx context.Context) ([]Job, error)
	FindByID(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
}
