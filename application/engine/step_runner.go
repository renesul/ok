package engine

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/database"
)

type StepRunner struct {
	planner  domain.Planner
	executor domain.Executor
	memory   *agentpkg.SQLiteMemory
	db       *sql.DB
}

func NewStepRunner(planner domain.Planner, executor domain.Executor, memory *agentpkg.SQLiteMemory, db *sql.DB) *StepRunner {
	return &StepRunner{
		planner:  planner,
		executor: executor,
		memory:   memory,
		db:       db,
	}
}

// ExecuteSingleStep trata fallback quando plano falha ou e vazio
func (s *StepRunner) ExecuteSingleStep(ctx context.Context, state *domain.ExecutionState, decision domain.Decision, input string, emitter Emitter, execStart time.Time) error {
	emitter.EmitStep(decision.Tool, decision.Tool, "running", 0)
	agentCtx := agentpkg.ToAgentContext(state)
	agentCtx.Add("user: " + input)
	plan, err := s.planner.Plan(decision, agentCtx)
	if errors.Is(err, agentpkg.ErrDone) || err != nil {
		if err != nil && !errors.Is(err, agentpkg.ErrDone) {
			emitter.EmitStep(decision.Tool, decision.Tool, "rejected", 0)
			emitter.EmitMessage(err.Error())
		} else {
			emitter.EmitMessage(decision.Input)
		}
		emitter.EmitDone()
		return nil
	}

	stepStart := time.Now()
	result, execErr := s.executor.Execute(plan)
	execMs := time.Since(stepStart).Milliseconds()

	if execErr != nil {
		emitter.EmitStep(plan.Tool.Name(), plan.Tool.Name(), "failed", execMs)
		emitter.EmitMessage(execErr.Error())
	} else {
		emitter.EmitStep(plan.Tool.Name(), plan.Tool.Name(), "done", execMs)
		emitter.EmitMessage(result)
		if s.memory != nil && s.db != nil && agentpkg.ShouldStore(input, result) {
			database.WithTx(s.db, ctx, func(tx *sql.Tx) error {
				return s.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: input + " -> " + result})
			})
		}
	}
	emitter.EmitDone()
	return nil
}

func LastStepOutput(state *domain.ExecutionState) string {
	if state.Plan == nil {
		return ""
	}
	// Prefer last successful output, fall back to last failed output
	for i := len(state.Plan.Steps) - 1; i >= 0; i-- {
		if state.Plan.Steps[i].Output != "" && state.Plan.Steps[i].Status == "done" {
			return state.Plan.Steps[i].Output
		}
	}
	for i := len(state.Plan.Steps) - 1; i >= 0; i-- {
		if state.Plan.Steps[i].Output != "" && state.Plan.Steps[i].Status == "failed" {
			return state.Plan.Steps[i].Output
		}
	}
	return ""
}
