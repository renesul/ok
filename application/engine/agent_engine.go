package engine

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

const (
	maxReplansPerStep   = 2
	contextPrunePercent = 0.8
)

// AgentEngine contem o loop unificado OBSERVE/PLAN/ACT/REFLECT
type AgentEngine struct {
	llmClient         *llm.Client
	llmConfig         llm.ClientConfig
	llmFastConfig     llm.ClientConfig
	planner           domain.Planner
	executor          domain.Executor
	limits            domain.AgentLimits
	buildSystemPrompt func() string
	log               *zap.Logger

	ctxManager      *ContextManager
	historyRecorder *HistoryRecorder
	stepRunner      *StepRunner
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
		llmClient:         llmClient,
		llmConfig:         llmConfig,
		llmFastConfig:     llmFastConfig,
		planner:           planner,
		executor:          executor,
		limits:            limits,
		buildSystemPrompt: buildSystemPrompt,
		log:               log.Named("engine"),

		ctxManager:      NewContextManager(llmConfig, buildSystemPrompt, log),
		historyRecorder: NewHistoryRecorder(db, memory, execRepo, log),
		stepRunner:      NewStepRunner(planner, executor, memory, db),
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
		MaxSteps:    e.limits.MaxSteps,
		MaxDuration: time.Duration(e.limits.TimeoutMs) * time.Millisecond,
	})
	emitter.EmitPhase("observe")

	memories, err := e.historyRecorder.SearchMemories(ctx, input, 5)
	if err == nil && len(memories) > 0 {
		var memStrings []string
		for _, m := range memories {
			state.Memories = append(state.Memories, m.Content)
			memStrings = append(memStrings, m.Content)
		}
		emitter.EmitMemories(memStrings)
	}

	agentpkg.AddEntry(state, domain.PhaseObserve, "user: "+input)
	systemPrompt := e.buildSystemPrompt()

	// Decide: conversa simples ou precisa de tools?
	const maxFastInputLen = 8000
	decideConfig := e.llmFastConfig
	if decideConfig.BaseURL == "" || len(input) > maxFastInputLen {
		decideConfig = e.llmConfig
	}
	decision, err := e.llmClient.Decide(ctx, decideConfig, systemPrompt, agentpkg.BuildContext(state))
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

	if decision.Tool != "" && decision.Done {
		decision.Done = false
		return e.stepRunner.ExecuteSingleStep(ctx, state, decision, input, emitter, execStart)
	}

	// PLAN
	emitter.EmitPhase("plan")
	agentpkg.Transition(state, domain.PhasePlan)
	e.ctxManager.PruneContextIfNeeded(ctx, state)

	planPrompt := agentpkg.BuildPlanningPrompt(input, e.planner.ToolDescriptions(), state.Memories)
	onPlanToken := func(token string) error {
		emitter.EmitStream("thought", token)
		return nil
	}
	executionPlan, planErr := e.llmClient.CreatePlanStreaming(ctx, e.llmConfig, planPrompt, input, onPlanToken)

	if planErr != nil || len(executionPlan.Steps) == 0 {
		return e.stepRunner.ExecuteSingleStep(ctx, state, decision, input, emitter, execStart)
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

		if st, ok := plan.Tool.(domain.StreamingTool); ok {
			st.SetStreamCallback(func(chunk string) {
				emitter.EmitStream(plan.Tool.Name(), chunk)
			})
			defer st.SetStreamCallback(nil)
		}

		stepStart := time.Now()
		result, execErr := e.executor.Execute(plan)
		execMs := time.Since(stepStart).Milliseconds()
		state.Attempts++

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
		e.ctxManager.PruneContextIfNeeded(ctx, state)
		stepOutput := e.ctxManager.SummarizeIfLong(ctx, step.Output)
		reflectPrompt := agentpkg.BuildReflectionPrompt(input, state.Plan.Steps[:state.CurrentStep+1], stepOutput)
		reflection, reflectErr := e.llmClient.Reflect(ctx, e.llmFastConfig, reflectPrompt, agentpkg.BuildContext(state))

		if reflectErr != nil {
			e.log.Error("reflection API failed or returned malformed JSON", zap.Error(reflectErr))
			agentpkg.AdvanceStep(state)
			replansThisStep = 0
			continue
		}

		agentpkg.AddEntry(state, domain.PhaseReflect, fmt.Sprintf("reflexao: %s - %s", reflection.Action, reflection.Reason))
		e.historyRecorder.ReflectAndLearn(ctx, state, step, reflection)

		switch reflection.Action {
		case "done":
			agentpkg.Transition(state, domain.PhaseDone)
			answer := reflection.FinalAnswer
			if answer == "" {
				answer = step.Output
			}
			emitter.EmitMessage(answer)
			e.historyRecorder.SaveResults(state, input, answer, execStart)
			emitter.EmitDone()
			return nil

		case "error":
			emitter.EmitMessage("erro: " + reflection.Reason)
			e.historyRecorder.SaveResults(state, input, "", execStart)
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

	agentpkg.Transition(state, domain.PhaseDone)
	lastOutput := LastStepOutput(state)
	if lastOutput != "" {
		emitter.EmitMessage(lastOutput)
	} else if budgetReason != "" {
		emitter.EmitMessage("execucao interrompida: " + budgetReason)
	} else {
		emitter.EmitMessage("execucao concluida")
	}

	e.historyRecorder.SaveResults(state, input, lastOutput, execStart)
	emitter.EmitDone()
	return nil
}
