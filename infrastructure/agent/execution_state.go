package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/renesul/ok/domain"
)

// Transicoes validas entre fases
var validTransitions = map[domain.ExecutionPhase][]domain.ExecutionPhase{
	domain.PhaseObserve: {domain.PhasePlan, domain.PhaseDone},
	domain.PhasePlan:    {domain.PhaseAct, domain.PhaseError},
	domain.PhaseAct:     {domain.PhaseReflect, domain.PhaseError},
	domain.PhaseReflect: {domain.PhaseAct, domain.PhasePlan, domain.PhaseDone, domain.PhaseError},
	domain.PhaseDone:    {},
	domain.PhaseError:   {},
}

func NewExecutionState(goal string, budget domain.ExecutionBudget) *domain.ExecutionState {
	budget.StartTime = time.Now()
	return &domain.ExecutionState{
		Goal:   goal,
		Phase:  domain.PhaseObserve,
		Budget: budget,
	}
}

// Transition valida e executa transicao de fase
func Transition(state *domain.ExecutionState, to domain.ExecutionPhase) error {
	allowed, ok := validTransitions[state.Phase]
	if !ok {
		return fmt.Errorf("phase unknown: %s", state.Phase)
	}
	for _, a := range allowed {
		if a == to {
			state.Phase = to
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s -> %s", state.Phase, to)
}

// IsBudgetExhausted verifica se o budget foi esgotado e retorna o motivo
func IsBudgetExhausted(state *domain.ExecutionState) (bool, string) {
	if state.Budget.MaxSteps > 0 && state.Attempts >= state.Budget.MaxSteps {
		return true, fmt.Sprintf("max steps atingido (%d)", state.Budget.MaxSteps)
	}
	if state.Budget.MaxDuration > 0 && time.Since(state.Budget.StartTime) >= state.Budget.MaxDuration {
		return true, fmt.Sprintf("timeout atingido (%s)", state.Budget.MaxDuration)
	}
	if state.Plan != nil && state.CurrentStep >= len(state.Plan.Steps) {
		return true, "plano completo"
	}
	return false, ""
}

func AddEntry(state *domain.ExecutionState, phase domain.ExecutionPhase, content string) {
	state.History = append(state.History, domain.ExecutionEntry{
		Phase:   phase,
		Content: content,
	})
}

func CurrentPlannedStep(state *domain.ExecutionState) *domain.PlannedStep {
	if state.Plan == nil || state.CurrentStep >= len(state.Plan.Steps) {
		return nil
	}
	return &state.Plan.Steps[state.CurrentStep]
}

func AdvanceStep(state *domain.ExecutionState) {
	state.CurrentStep++
}

func SetPlan(state *domain.ExecutionState, plan domain.ExecutionPlan) {
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}
	state.Plan = &plan
	state.CurrentStep = 0
}

func ReplacePlan(state *domain.ExecutionState, remaining []domain.PlannedStep) {
	if state.Plan == nil {
		return
	}
	for i := range remaining {
		remaining[i].Status = "pending"
	}
	completed := state.Plan.Steps[:state.CurrentStep]
	state.Plan.Steps = append(completed, remaining...)
}

// FailedAttempt — registro de tentativa falhada para detectar loops
type FailedAttempt struct {
	Tool  string
	Input string
}

// ExtractFailedAttempts — extrai steps que falharam do plano atual
func ExtractFailedAttempts(state *domain.ExecutionState) []FailedAttempt {
	if state.Plan == nil {
		return nil
	}
	var failed []FailedAttempt
	for _, step := range state.Plan.Steps[:state.CurrentStep] {
		if step.Status == "failed" {
			failed = append(failed, FailedAttempt{Tool: step.Tool, Input: step.Input})
		}
	}
	return failed
}

// FilterRepeatedSteps — remove steps que repetem tentativas falhadas (mesmo tool+input)
func FilterRepeatedSteps(newSteps []domain.PlannedStep, failed []FailedAttempt) []domain.PlannedStep {
	if len(failed) == 0 {
		return newSteps
	}
	failedSet := make(map[string]bool)
	for _, f := range failed {
		failedSet[f.Tool+":"+f.Input] = true
	}
	var filtered []domain.PlannedStep
	for _, step := range newSteps {
		if !failedSet[step.Tool+":"+step.Input] {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

// ReplacePlanSafe — replan com protecao contra loops
// Retorna true se o novo plano foi aplicado, false se todos os steps foram filtrados
func ReplacePlanSafe(state *domain.ExecutionState, remaining []domain.PlannedStep) bool {
	if state.Plan == nil || len(remaining) == 0 {
		return false
	}

	failed := ExtractFailedAttempts(state)
	filtered := FilterRepeatedSteps(remaining, failed)

	if len(filtered) == 0 {
		return false
	}

	ReplacePlan(state, filtered)
	return true
}

func BuildContext(state *domain.ExecutionState) string {
	var parts []string
	parts = append(parts, "Objetivo: "+state.Goal)

	if len(state.Memories) > 0 {
		parts = append(parts, "Memorias relevantes:\n"+strings.Join(state.Memories, "\n"))
	}

	if state.Plan != nil {
		var planDesc []string
		for i, step := range state.Plan.Steps {
			marker := "[ ]"
			if step.Status == "done" {
				marker = "[x]"
			} else if step.Status == "failed" {
				marker = "[!]"
			}
			line := fmt.Sprintf("%s %d. %s (tool: %s)", marker, i+1, step.Name, step.Tool)
			if step.Output != "" {
				line += " -> " + TruncateWithEllipsis(step.Output, 200)
			}
			planDesc = append(planDesc, line)
		}
		parts = append(parts, "Plano:\n"+strings.Join(planDesc, "\n"))
	}

	for _, entry := range state.History {
		parts = append(parts, fmt.Sprintf("[%s] %s", entry.Phase, entry.Content))
	}

	return strings.Join(parts, "\n\n")
}

func ToAgentContext(state *domain.ExecutionState) *domain.AgentContext {
	ctx := domain.NewAgentContext(state.Budget.MaxSteps)
	for _, entry := range state.History {
		ctx.Add(fmt.Sprintf("%s: %s", entry.Phase, entry.Content))
	}
	ctx.Steps = state.Attempts
	return ctx
}

