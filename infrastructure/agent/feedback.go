package agent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FeedbackRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewFeedbackRepository(db *gorm.DB, log *zap.Logger) *FeedbackRepository {
	return &FeedbackRepository{db: db, log: log.Named("agent.feedback")}
}

func (r *FeedbackRepository) Save(ctx context.Context, feedback *domain.Feedback) error {
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now()
	}

	r.log.Debug("save feedback", zap.String("tool", feedback.ToolName), zap.Bool("success", feedback.Success))

	return r.db.WithContext(ctx).Exec(
		"INSERT INTO agent_feedback (id, tool_name, task_type, success, duration_ms, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		feedback.ID, feedback.ToolName, feedback.TaskType, feedback.Success, feedback.DurationMs, feedback.Error, feedback.CreatedAt,
	).Error
}

