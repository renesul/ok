package agent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type FeedbackRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewFeedbackRepository(db *sql.DB, log *zap.Logger) *FeedbackRepository {
	return &FeedbackRepository{db: db, log: log.Named("agent.feedback")}
}

func (r *FeedbackRepository) Save(ctx context.Context, feedback *domain.Feedback) error {
	ctx = database.Ctx(ctx)
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now()
	}

	r.log.Debug("save feedback", zap.String("tool", feedback.ToolName), zap.Bool("success", feedback.Success))

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO agent_feedback (id, tool_name, task_type, success, duration_ms, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		feedback.ID, feedback.ToolName, feedback.TaskType, feedback.Success, feedback.DurationMs, feedback.Error, feedback.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save feedback: %w", err)
	}
	return nil
}
