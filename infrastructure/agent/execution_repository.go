package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ExecutionRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewExecutionRepository(db *gorm.DB, log *zap.Logger) *ExecutionRepository {
	return &ExecutionRepository{db: db, log: log.Named("agent.execution")}
}

func (r *ExecutionRepository) Save(record *domain.ExecutionRecord) error {
	return r.SaveInTx(r.db, record)
}

// SaveInTx salva um execution record dentro de uma transacao existente
func (r *ExecutionRepository) SaveInTx(tx *gorm.DB, record *domain.ExecutionRecord) error {
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

	return tx.Exec(
		"INSERT INTO agent_executions (id, goal, status, steps, timeline, total_ms, step_count, tools_used, failure_reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		record.ID, record.Goal, record.Status, string(stepsJSON), string(timelineJSON), record.TotalMs, record.StepCount, string(toolsUsedJSON), record.FailureReason, record.CreatedAt,
	).Error
}

func (r *ExecutionRepository) FindByID(id string) (*domain.ExecutionRecord, error) {
	var row struct {
		ID            string
		Goal          string
		Status        string
		Steps         string
		Timeline      string
		TotalMs       int64
		StepCount     int
		ToolsUsed     string
		FailureReason string
		CreatedAt     time.Time
	}

	err := r.db.Raw("SELECT id, goal, status, steps, timeline, total_ms, step_count, COALESCE(tools_used,'') as tools_used, COALESCE(failure_reason,'') as failure_reason, created_at FROM agent_executions WHERE id = ?", id).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("find execution: %w", err)
	}
	if row.ID == "" {
		return nil, nil
	}

	record := &domain.ExecutionRecord{
		ID:            row.ID,
		Goal:          row.Goal,
		Status:        row.Status,
		TotalMs:       row.TotalMs,
		StepCount:     row.StepCount,
		FailureReason: row.FailureReason,
		CreatedAt:     row.CreatedAt,
	}

	if err := json.Unmarshal([]byte(row.Steps), &record.Steps); err != nil {
		r.log.Debug("unmarshal steps failed", zap.String("id", row.ID), zap.Error(err))
	}
	if err := json.Unmarshal([]byte(row.Timeline), &record.Timeline); err != nil {
		r.log.Debug("unmarshal timeline failed", zap.String("id", row.ID), zap.Error(err))
	}
	if err := json.Unmarshal([]byte(row.ToolsUsed), &record.ToolsUsed); err != nil {
		r.log.Debug("unmarshal tools_used failed", zap.String("id", row.ID), zap.Error(err))
	}

	return record, nil
}

func (r *ExecutionRepository) FindRecent(limit int) ([]domain.ExecutionRecord, error) {
	var rows []struct {
		ID            string
		Goal          string
		Status        string
		Steps         string
		Timeline      string
		TotalMs       int64
		StepCount     int
		ToolsUsed     string
		FailureReason string
		CreatedAt     time.Time
	}

	err := r.db.Raw("SELECT id, goal, status, steps, timeline, total_ms, step_count, COALESCE(tools_used,'') as tools_used, COALESCE(failure_reason,'') as failure_reason, created_at FROM agent_executions ORDER BY created_at DESC LIMIT ?", limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find recent executions: %w", err)
	}

	records := make([]domain.ExecutionRecord, len(rows))
	for i, row := range rows {
		records[i] = domain.ExecutionRecord{
			ID:            row.ID,
			Goal:          row.Goal,
			Status:        row.Status,
			TotalMs:       row.TotalMs,
			StepCount:     row.StepCount,
			FailureReason: row.FailureReason,
			CreatedAt:     row.CreatedAt,
		}
		if err := json.Unmarshal([]byte(row.Steps), &records[i].Steps); err != nil {
			r.log.Debug("unmarshal steps failed", zap.String("id", row.ID), zap.Error(err))
		}
		if err := json.Unmarshal([]byte(row.Timeline), &records[i].Timeline); err != nil {
			r.log.Debug("unmarshal timeline failed", zap.String("id", row.ID), zap.Error(err))
		}
		if err := json.Unmarshal([]byte(row.ToolsUsed), &records[i].ToolsUsed); err != nil {
			r.log.Debug("unmarshal tools_used failed", zap.String("id", row.ID), zap.Error(err))
		}
	}

	return records, nil
}

func (r *ExecutionRepository) GetMetrics() (*domain.ExecutionMetrics, error) {
	metrics := &domain.ExecutionMetrics{
		ToolUsageCount: make(map[string]int),
	}

	var stats struct {
		Total       int
		SuccessRate float64
		AvgMs       int64
		AvgSteps    float64
	}
	err := r.db.Raw(`
		SELECT COUNT(*) as total,
			AVG(CASE WHEN status = 'done' THEN 1.0 ELSE 0.0 END) as success_rate,
			CAST(AVG(total_ms) AS INTEGER) as avg_ms,
			AVG(step_count) as avg_steps
		FROM agent_executions
	`).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("get execution metrics: %w", err)
	}

	metrics.TotalExecutions = stats.Total
	metrics.SuccessRate = stats.SuccessRate
	metrics.AvgDurationMs = stats.AvgMs
	metrics.AvgStepCount = stats.AvgSteps

	// Tool usage from recent executions
	var rows []struct{ ToolsUsed string }
	r.db.Raw("SELECT COALESCE(tools_used,'') as tools_used FROM agent_executions WHERE tools_used != '' ORDER BY created_at DESC LIMIT 100").Scan(&rows)
	for _, row := range rows {
		var tools []string
		json.Unmarshal([]byte(row.ToolsUsed), &tools)
		for _, tool := range tools {
			metrics.ToolUsageCount[tool]++
		}
	}

	return metrics, nil
}

