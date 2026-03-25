package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type ExecutionRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewExecutionRepository(db *sql.DB, log *zap.Logger) *ExecutionRepository {
	return &ExecutionRepository{db: db, log: log.Named("agent.execution")}
}

func (r *ExecutionRepository) Save(record *domain.ExecutionRecord) error {
	return r.SaveInTx(r.db, record)
}

// SaveInTx salva um execution record dentro de uma transacao existente
func (r *ExecutionRepository) SaveInTx(tx database.Execer, record *domain.ExecutionRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	stepsJSON, err := json.Marshal(record.Steps)
	if err != nil {
		stepsJSON = []byte("[]")
	}
	timelineJSON, err := json.Marshal(record.Timeline)
	if err != nil {
		timelineJSON = []byte("[]")
	}
	toolsUsedJSON, err := json.Marshal(record.ToolsUsed)
	if err != nil {
		toolsUsedJSON = []byte("[]")
	}

	_, err = tx.ExecContext(context.Background(),
		"INSERT INTO agent_executions (id, goal, status, steps, timeline, total_ms, step_count, tools_used, failure_reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		record.ID, record.Goal, record.Status, string(stepsJSON), string(timelineJSON), record.TotalMs, record.StepCount, string(toolsUsedJSON), record.FailureReason, record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save execution: %w", err)
	}
	return nil
}

func (r *ExecutionRepository) FindByID(id string) (*domain.ExecutionRecord, error) {
	row := r.db.QueryRow(
		"SELECT id, goal, status, steps, timeline, total_ms, step_count, COALESCE(tools_used,'') as tools_used, COALESCE(failure_reason,'') as failure_reason, created_at FROM agent_executions WHERE id = ?", id,
	)

	var rec struct {
		ID, Goal, Status, Steps, Timeline, ToolsUsed, FailureReason string
		TotalMs                                                      int64
		StepCount                                                    int
		CreatedAt                                                    time.Time
	}
	err := row.Scan(&rec.ID, &rec.Goal, &rec.Status, &rec.Steps, &rec.Timeline, &rec.TotalMs, &rec.StepCount, &rec.ToolsUsed, &rec.FailureReason, &rec.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find execution: %w", err)
	}

	record := &domain.ExecutionRecord{
		ID:            rec.ID,
		Goal:          rec.Goal,
		Status:        rec.Status,
		TotalMs:       rec.TotalMs,
		StepCount:     rec.StepCount,
		FailureReason: rec.FailureReason,
		CreatedAt:     rec.CreatedAt,
	}

	if err := json.Unmarshal([]byte(rec.Steps), &record.Steps); err != nil {
		r.log.Debug("unmarshal steps failed", zap.String("id", rec.ID), zap.Error(err))
	}
	if err := json.Unmarshal([]byte(rec.Timeline), &record.Timeline); err != nil {
		r.log.Debug("unmarshal timeline failed", zap.String("id", rec.ID), zap.Error(err))
	}
	if err := json.Unmarshal([]byte(rec.ToolsUsed), &record.ToolsUsed); err != nil {
		r.log.Debug("unmarshal tools_used failed", zap.String("id", rec.ID), zap.Error(err))
	}

	return record, nil
}

func (r *ExecutionRepository) FindRecent(limit int) ([]domain.ExecutionRecord, error) {
	rows, err := r.db.Query(
		"SELECT id, goal, status, steps, timeline, total_ms, step_count, COALESCE(tools_used,'') as tools_used, COALESCE(failure_reason,'') as failure_reason, created_at FROM agent_executions ORDER BY created_at DESC LIMIT ?", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find recent executions: %w", err)
	}
	defer rows.Close()

	var records []domain.ExecutionRecord
	for rows.Next() {
		var id, goal, status, steps, timeline, toolsUsed, failureReason string
		var totalMs int64
		var stepCount int
		var createdAt time.Time

		if err := rows.Scan(&id, &goal, &status, &steps, &timeline, &totalMs, &stepCount, &toolsUsed, &failureReason, &createdAt); err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}

		rec := domain.ExecutionRecord{
			ID:            id,
			Goal:          goal,
			Status:        status,
			TotalMs:       totalMs,
			StepCount:     stepCount,
			FailureReason: failureReason,
			CreatedAt:     createdAt,
		}
		if err := json.Unmarshal([]byte(steps), &rec.Steps); err != nil {
			r.log.Debug("unmarshal steps failed", zap.String("id", id), zap.Error(err))
		}
		if err := json.Unmarshal([]byte(timeline), &rec.Timeline); err != nil {
			r.log.Debug("unmarshal timeline failed", zap.String("id", id), zap.Error(err))
		}
		if err := json.Unmarshal([]byte(toolsUsed), &rec.ToolsUsed); err != nil {
			r.log.Debug("unmarshal tools_used failed", zap.String("id", id), zap.Error(err))
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

func (r *ExecutionRepository) GetMetrics() (*domain.ExecutionMetrics, error) {
	metrics := &domain.ExecutionMetrics{
		ToolUsageCount: make(map[string]int),
	}

	err := r.db.QueryRow(`
		SELECT COUNT(*) as total,
			COALESCE(AVG(CASE WHEN status = 'done' THEN 1.0 ELSE 0.0 END), 0) as success_rate,
			COALESCE(CAST(AVG(total_ms) AS INTEGER), 0) as avg_ms,
			COALESCE(AVG(step_count), 0) as avg_steps
		FROM agent_executions
	`).Scan(&metrics.TotalExecutions, &metrics.SuccessRate, &metrics.AvgDurationMs, &metrics.AvgStepCount)
	if err != nil {
		return nil, fmt.Errorf("get execution metrics: %w", err)
	}

	// Tool usage from recent executions
	rows, err := r.db.Query("SELECT COALESCE(tools_used,'') FROM agent_executions WHERE tools_used != '' ORDER BY created_at DESC LIMIT 100")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var toolsUsed string
			if err := rows.Scan(&toolsUsed); err == nil {
				var tools []string
				json.Unmarshal([]byte(toolsUsed), &tools)
				for _, tool := range tools {
					metrics.ToolUsageCount[tool]++
				}
			}
		}
	}

	return metrics, nil
}
