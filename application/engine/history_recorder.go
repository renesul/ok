package engine

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type HistoryRecorder struct {
	db       *sql.DB
	memory   *agentpkg.SQLiteMemory
	execRepo *agentpkg.ExecutionRepository
	log      *zap.Logger
}

func NewHistoryRecorder(db *sql.DB, memory *agentpkg.SQLiteMemory, execRepo *agentpkg.ExecutionRepository, log *zap.Logger) *HistoryRecorder {
	return &HistoryRecorder{
		db:       db,
		memory:   memory,
		execRepo: execRepo,
		log:      log.Named("history_recorder"),
	}
}

// SearchMemories busca memorias semanticas relevantes para o input.
func (h *HistoryRecorder) SearchMemories(ctx context.Context, input string, limit int) ([]domain.MemoryEntry, error) {
	if h.memory == nil {
		return nil, nil
	}
	return h.memory.SearchSemantic(ctx, input, limit)
}

// SaveResults persiste execution record + memoria em uma unica transacao
func (h *HistoryRecorder) SaveResults(state *domain.ExecutionState, input, output string, startTime time.Time) {
	if h.db == nil {
		return
	}

	err := database.WithTx(h.db, context.Background(), func(tx *sql.Tx) error {
		// Salvar execution record
		if h.execRepo != nil {
			record := h.buildExecutionRecord(state, startTime)
			if err := h.execRepo.SaveInTx(tx, record); err != nil {
				return fmt.Errorf("save execution: %w", err)
			}
		}

		// Salvar memoria se relevante
		if h.memory != nil && agentpkg.ShouldStore(input, output) {
			entry := domain.MemoryEntry{
				Content: input + " -> " + output,
			}
			if err := h.memory.SaveChunkedInTx(tx, entry); err != nil {
				return fmt.Errorf("save memory: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		h.log.Debug("save results failed", zap.Error(err))
	}
}

func (h *HistoryRecorder) ReflectAndLearn(ctx context.Context, state *domain.ExecutionState, step *domain.PlannedStep, reflection domain.ReflectionResult) {
	if h.memory == nil || h.db == nil {
		return
	}

	database.WithTx(h.db, context.Background(), func(tx *sql.Tx) error {
		memContent := fmt.Sprintf("%s: %s -> %s [%s]",
			step.Tool, step.Input, step.Output, step.Status)
		if err := h.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: memContent}); err != nil {
			return fmt.Errorf("save reflection memory: %w", err)
		}

		if reflection.Action == "error" || reflection.Action == "replan" {
			failureMemory := fmt.Sprintf("reflexao:%s falhou em '%s' - %s",
				step.Tool, step.Input, reflection.Reason)
			if err := h.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: failureMemory}); err != nil {
				return fmt.Errorf("save failure memory: %w", err)
			}
		}
		return nil
	})
}

func (h *HistoryRecorder) buildExecutionRecord(state *domain.ExecutionState, startTime time.Time) *domain.ExecutionRecord {
	status := "done"
	if state.Phase == domain.PhaseError {
		status = "error"
	}

	toolSet := make(map[string]bool)
	if state.Plan != nil {
		for _, step := range state.Plan.Steps {
			if step.Tool != "" && step.Status == "done" {
				toolSet[step.Tool] = true
			}
		}
	}
	var toolsUsed []string
	for tool := range toolSet {
		toolsUsed = append(toolsUsed, tool)
	}

	var failureReason string
	if state.Plan != nil {
		for i := len(state.Plan.Steps) - 1; i >= 0; i-- {
			if state.Plan.Steps[i].Status == "failed" {
				failureReason = "step '" + state.Plan.Steps[i].Name + "' failed (tool: " + state.Plan.Steps[i].Tool + ")"
				break
			}
		}
	}

	return &domain.ExecutionRecord{
		Goal:          state.Goal,
		Status:        status,
		Steps:         h.extractStepResults(state),
		Timeline:      state.History,
		TotalMs:       time.Since(startTime).Milliseconds(),
		StepCount:     len(toolsUsed),
		ToolsUsed:     toolsUsed,
		FailureReason: failureReason,
	}
}

func (h *HistoryRecorder) extractStepResults(state *domain.ExecutionState) []domain.StepResult {
	if state.Plan == nil {
		return nil
	}
	var results []domain.StepResult
	for _, step := range state.Plan.Steps {
		if step.Status != "pending" && step.Status != "" {
			results = append(results, domain.StepResult{
				Name:   step.Name,
				Tool:   step.Tool,
				Status: step.Status,
			})
		}
	}
	return results
}
