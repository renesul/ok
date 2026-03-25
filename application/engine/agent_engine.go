package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

const (
	maxReplansPerStep    = 2
	contextPrunePercent  = 0.8
)

// AgentEngine contem o loop unificado OBSERVE/PLAN/ACT/REFLECT
type AgentEngine struct {
	db                *sql.DB
	llmClient         *llm.Client
	llmConfig         llm.ClientConfig
	llmFastConfig     llm.ClientConfig
	planner           domain.Planner
	executor          domain.Executor
	memory            *agentpkg.SQLiteMemory
	execRepo          *agentpkg.ExecutionRepository
	limits            domain.AgentLimits
	buildSystemPrompt func() string
	log               *zap.Logger
}

func NewAgentEngine(
	db *sql.DB,
	llmClient *llm.Client,
	llmConfig llm.ClientConfig,
	llmFastConfig llm.ClientConfig,
	planner domain.Planner,
	executor domain.Executor,
	memory *agentpkg.SQLiteMemory,
	execRepo *agentpkg.ExecutionRepository,
	limits domain.AgentLimits,
	buildSystemPrompt func() string,
	log *zap.Logger,
) *AgentEngine {
	return &AgentEngine{
		db:                db,
		llmClient:         llmClient,
		llmConfig:         llmConfig,
		llmFastConfig:     llmFastConfig,
		planner:           planner,
		executor:          executor,
		memory:            memory,
		execRepo:          execRepo,
		limits:            limits,
		buildSystemPrompt: buildSystemPrompt,
		log:               log.Named("engine"),
	}
}

// RunLoop executa o loop autonomo OBSERVE/PLAN/ACT/REFLECT uma unica vez
func (e *AgentEngine) RunLoop(ctx context.Context, input string, emitter Emitter) error {
	execStart := time.Now()

	// Reset delegate counter per execution
	if dt, ok := e.planner.Tools()["delegate"]; ok {
		if resettable, ok := dt.(interface{ ResetCount() }); ok {
			resettable.ResetCount()
		}
	}

	// OBSERVE
	state := agentpkg.NewExecutionState(input, domain.ExecutionBudget{
		MaxSteps:    e.limits.MaxAttempts,
		MaxDuration: time.Duration(e.limits.TimeoutMs) * time.Millisecond,
	})
	emitter.EmitPhase("observe")

	if e.memory != nil {
		memories, err := e.memory.SearchSemantic(ctx, input, 5)
		if err == nil && len(memories) > 0 {
			var memStrings []string
			for _, m := range memories {
				state.Memories = append(state.Memories, m.Content)
				memStrings = append(memStrings, m.Content)
			}
			emitter.EmitMemories(memStrings)
		}
	}

	agentpkg.AddEntry(state, domain.PhaseObserve, "user: "+input)
	systemPrompt := e.buildSystemPrompt()

	// Decide: conversa simples ou precisa de tools?
	// Usa modelo fast se disponivel, mas pula para pesado se input muito grande
	const maxFastInputLen = 8000
	decideConfig := e.llmFastConfig
	if decideConfig.BaseURL == "" || len(input) > maxFastInputLen {
		decideConfig = e.llmConfig
	}
	decision, err := e.llmClient.Decide(ctx, decideConfig, systemPrompt, agentpkg.BuildContext(state))
	state.Attempts++
	if err != nil {
		emitter.EmitMessage("error: " + err.Error())
		emitter.EmitDone()
		return nil
	}

	if decision.Done && decision.Tool == "" {
		emitter.EmitMessage(decision.Input)
		emitter.EmitDone()
		return nil
	}

	// LLM quer usar tool (mesmo com done=true) — executar diretamente
	if decision.Tool != "" && decision.Done {
		decision.Done = false // planner rejeita Done=true
		return e.executeSingleStep(ctx, state, decision, input, emitter, execStart)
	}

	// PLAN
	emitter.EmitPhase("plan")
	agentpkg.Transition(state, domain.PhasePlan)
	e.pruneContextIfNeeded(ctx, state)

	planPrompt := agentpkg.BuildPlanningPrompt(input, e.planner.ToolDescriptions(), state.Memories)
	onPlanToken := func(token string) error {
		emitter.EmitStream("thought", token)
		return nil
	}
	executionPlan, planErr := e.llmClient.CreatePlanStreaming(ctx, e.llmConfig, planPrompt, input, onPlanToken)
	state.Attempts++

	if planErr != nil || len(executionPlan.Steps) == 0 {
		return e.executeSingleStep(ctx, state, decision, input, emitter, execStart)
	}

	agentpkg.SetPlan(state, executionPlan)
	agentpkg.AddEntry(state, domain.PhasePlan, fmt.Sprintf("plano com %d passos: %s", len(executionPlan.Steps), executionPlan.Reasoning))
	emitter.EmitMessage(executionPlan.Reasoning)

	// ACT + REFLECT loop
	replansThisStep := 0
	var budgetReason string
	for exhausted, reason := agentpkg.IsBudgetExhausted(state); !exhausted; exhausted, reason = agentpkg.IsBudgetExhausted(state) {
		budgetReason = reason
		step := agentpkg.CurrentPlannedStep(state)
		if step == nil {
			break
		}

		emitter.EmitPhase("act")
		emitter.EmitStep(step.Name, step.Tool, "running", 0)

		stepDecision := domain.Decision{Tool: step.Tool, Input: step.Input}
		agentCtx := agentpkg.ToAgentContext(state)
		plan, planErr := e.planner.Plan(stepDecision, agentCtx)
		if planErr != nil {
			step.Status = "failed"
			emitter.EmitStep(step.Name, step.Tool, "failed", 0)
			agentpkg.AdvanceStep(state)
			continue
		}

		// Injetar stream callback se tool suporta streaming (PTY)
		if st, ok := plan.Tool.(domain.StreamingTool); ok {
			st.SetStreamCallback(func(chunk string) {
				emitter.EmitStream(plan.Tool.Name(), chunk)
			})
			defer st.SetStreamCallback(nil)
		}

		stepStart := time.Now()
		result, execErr := e.executor.Execute(plan)
		execMs := time.Since(stepStart).Milliseconds()

		if execErr != nil {
			step.Status = "failed"
			step.Output = "error: " + execErr.Error()
			emitter.EmitStep(step.Name, step.Tool, "failed", execMs)
		} else {
			step.Status = "done"
			step.Output = result
			emitter.EmitStep(step.Name, step.Tool, "done", execMs)
		}

		// REFLECT
		emitter.EmitPhase("reflect")
		agentpkg.Transition(state, domain.PhaseReflect)
		e.pruneContextIfNeeded(ctx, state)
		stepOutput := e.summarizeIfLong(ctx, step.Output)
		reflectPrompt := agentpkg.BuildReflectionPrompt(input, state.Plan.Steps[:state.CurrentStep+1], stepOutput)
		reflection, reflectErr := e.llmClient.Reflect(ctx, e.llmFastConfig, reflectPrompt, agentpkg.BuildContext(state))
		state.Attempts++

		if reflectErr != nil {
			e.log.Error("reflection API failed or returned malformed JSON", zap.Error(reflectErr))
			agentpkg.AdvanceStep(state)
			replansThisStep = 0
			continue
		}

		agentpkg.AddEntry(state, domain.PhaseReflect, fmt.Sprintf("reflexao: %s - %s", reflection.Action, reflection.Reason))
		e.reflectAndLearn(ctx, state, step, reflection)

		switch reflection.Action {
		case "done":
			agentpkg.Transition(state, domain.PhaseDone)
			answer := reflection.FinalAnswer
			if answer == "" {
				answer = step.Output
			}
			emitter.EmitMessage(answer)
			e.saveResults(state, input, answer, execStart)
			emitter.EmitDone()
			return nil

		case "error":
			emitter.EmitMessage("erro: " + reflection.Reason)
			e.saveResults(state, input, "", execStart)
			emitter.EmitDone()
			return nil

		case "replan":
			if replansThisStep < maxReplansPerStep && len(reflection.RevisedPlan) > 0 {
				agentpkg.AdvanceStep(state)
				if agentpkg.ReplacePlanSafe(state, reflection.RevisedPlan) {
					replansThisStep++
				} else {
					replansThisStep = 0
				}
			} else {
				agentpkg.AdvanceStep(state)
				replansThisStep = 0
			}

		default:
			agentpkg.AdvanceStep(state)
			replansThisStep = 0
		}
	}

	// Plano completado ou budget esgotado
	agentpkg.Transition(state, domain.PhaseDone)
	lastOutput := e.lastStepOutput(state)
	if lastOutput != "" {
		emitter.EmitMessage(lastOutput)
	} else if budgetReason != "" {
		emitter.EmitMessage("execucao interrompida: " + budgetReason)
	} else {
		emitter.EmitMessage("execucao concluida")
	}

	e.saveResults(state, input, lastOutput, execStart)
	emitter.EmitDone()
	return nil
}

// executeSingleStep trata fallback quando plano falha ou e vazio
func (e *AgentEngine) executeSingleStep(ctx context.Context, state *domain.ExecutionState, decision domain.Decision, input string, emitter Emitter, execStart time.Time) error {
	emitter.EmitStep(decision.Tool, decision.Tool, "running", 0)
	agentCtx := agentpkg.ToAgentContext(state)
	agentCtx.Add("user: " + input)
	plan, err := e.planner.Plan(decision, agentCtx)
	if errors.Is(err, agentpkg.ErrDone) || err != nil {
		if err != nil && !errors.Is(err, agentpkg.ErrDone) {
			emitter.EmitStep(decision.Tool, decision.Tool, "rejected", 0)
			emitter.EmitMessage("planner: " + err.Error())
		} else {
			emitter.EmitMessage(decision.Input)
		}
		emitter.EmitDone()
		return nil
	}

	stepStart := time.Now()
	result, execErr := e.executor.Execute(plan)
	execMs := time.Since(stepStart).Milliseconds()

	if execErr != nil {
		emitter.EmitStep(plan.Tool.Name(), plan.Tool.Name(), "failed", execMs)
		emitter.EmitMessage("execution: " + execErr.Error())
	} else {
		emitter.EmitStep(plan.Tool.Name(), plan.Tool.Name(), "done", execMs)
		emitter.EmitMessage(result)
		if e.memory != nil && e.db != nil && agentpkg.ShouldStore(input, result) {
			database.WithTx(e.db, ctx, func(tx *sql.Tx) error {
				return e.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: input + " -> " + result})
			})
		}
	}
	emitter.EmitDone()
	return nil
}

func (e *AgentEngine) lastStepOutput(state *domain.ExecutionState) string {
	if state.Plan == nil {
		return ""
	}
	for i := len(state.Plan.Steps) - 1; i >= 0; i-- {
		if state.Plan.Steps[i].Output != "" && state.Plan.Steps[i].Status == "done" {
			return state.Plan.Steps[i].Output
		}
	}
	return ""
}

// saveResults persiste execution record + memoria em uma unica transacao
func (e *AgentEngine) saveResults(state *domain.ExecutionState, input, output string, startTime time.Time) {
	if e.db == nil {
		return
	}

	err := database.WithTx(e.db, context.Background(), func(tx *sql.Tx) error {
		// Salvar execution record
		if e.execRepo != nil {
			record := e.buildExecutionRecord(state, startTime)
			if err := e.execRepo.SaveInTx(tx, record); err != nil {
				return fmt.Errorf("save execution: %w", err)
			}
		}

		// Salvar memoria se relevante
		if e.memory != nil && agentpkg.ShouldStore(input, output) {
			entry := domain.MemoryEntry{
				Content: input + " -> " + output,
			}
			if err := e.memory.SaveChunkedInTx(tx, entry); err != nil {
				return fmt.Errorf("save memory: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		e.log.Debug("save results failed", zap.Error(err))
	}
}

func (e *AgentEngine) reflectAndLearn(ctx context.Context, state *domain.ExecutionState, step *domain.PlannedStep, reflection domain.ReflectionResult) {
	if e.memory == nil || e.db == nil {
		return
	}

	database.WithTx(e.db, context.Background(), func(tx *sql.Tx) error {
		memContent := fmt.Sprintf("%s: %s -> %s [%s]",
			step.Tool, step.Input, step.Output, step.Status)
		if err := e.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: memContent}); err != nil {
			return fmt.Errorf("save reflection memory: %w", err)
		}

		if reflection.Action == "error" || reflection.Action == "replan" {
			failureMemory := fmt.Sprintf("reflexao:%s falhou em '%s' - %s",
				step.Tool, step.Input, reflection.Reason)
			if err := e.memory.SaveChunkedInTx(tx, domain.MemoryEntry{Content: failureMemory}); err != nil {
				return fmt.Errorf("save failure memory: %w", err)
			}
		}
		return nil
	})
}

func (e *AgentEngine) buildExecutionRecord(state *domain.ExecutionState, startTime time.Time) *domain.ExecutionRecord {
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
		Steps:         e.extractStepResults(state),
		Timeline:      state.History,
		TotalMs:       time.Since(startTime).Milliseconds(),
		StepCount:     len(toolsUsed),
		ToolsUsed:     toolsUsed,
		FailureReason: failureReason,
	}
}

func (e *AgentEngine) extractStepResults(state *domain.ExecutionState) []domain.StepResult {
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

// estimateTokens aproxima contagem de tokens via chars/4
func estimateTokens(text string) int {
	return len(text) / 4
}

// pruneContextIfNeeded descarta metade antiga do historico quando o contexto
// ultrapassa 80% da janela de tokens. Sliding window simples — sem chamada LLM.
func (e *AgentEngine) pruneContextIfNeeded(_ context.Context, state *domain.ExecutionState) {
	maxTokens := e.llmConfig.MaxContextTokens
	if maxTokens <= 0 {
		return
	}

	systemPrompt := e.buildSystemPrompt()
	ctx := agentpkg.BuildContext(state)
	totalTokens := estimateTokens(systemPrompt) + estimateTokens(ctx)
	threshold := int(float64(maxTokens) * contextPrunePercent)

	if totalTokens < threshold {
		return
	}

	historyLen := len(state.History)
	if historyLen < 4 {
		return
	}
	splitAt := historyLen / 2

	synthEntry := domain.ExecutionEntry{
		Phase:   domain.PhaseObserve,
		Content: fmt.Sprintf("[Contexto antigo podado — %d entradas removidas]", splitAt),
	}
	state.History = append([]domain.ExecutionEntry{synthEntry}, state.History[splitAt:]...)

	e.log.Debug("context pruned",
		zap.Int("old_entries", splitAt),
		zap.Int("new_total", len(state.History)),
		zap.Int("tokens_before", totalTokens),
	)
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// summarizeIfLong was previously a synchronous LLM call that caused N+1 API bottlenecks.
// It is now an O(1) pure string truncation to preserve extreme OODA loop velocity.
func (e *AgentEngine) summarizeIfLong(ctx context.Context, output string) string {
	const maxOutputLength = 1500
	if len(output) <= maxOutputLength {
		return output
	}
	
	e.log.Debug("truncating long tool output to prevent window exhaustion", zap.Int("len", len(output)))
	return agentpkg.TruncateWithEllipsis(output, maxOutputLength)
}
