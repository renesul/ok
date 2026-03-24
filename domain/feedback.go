package domain

import (
	"context"
	"time"
)

type Feedback struct {
	ID         string
	ToolName   string
	TaskType   string
	Success    bool
	DurationMs int64
	Error      string
	CreatedAt  time.Time
}

type FeedbackRepository interface {
	Save(ctx context.Context, feedback *Feedback) error
}
