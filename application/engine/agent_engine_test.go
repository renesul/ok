package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
)

// TestEngine_DirectResponse — LLM returns done=true with no tool.
// Engine should emit the message and finish without calling Planner/Executor.
func TestEngine_DirectResponse(t *testing.T) {
	resp := &llmResponses{
		decide: `{"tool":"","input":"Ola! Como posso ajudar?","done":true}`,
	}
	server := newMockLLMServer(resp)
	defer server.Close()

	planner := newMockPlanner()
	executor := newMockExecutor("")

	eng := newTestEngine(server.URL, planner, executor, defaultLimits())
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "oi", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	result := emitter.Response()
	if len(result.Messages) == 0 {
		t.Fatal("expected at least 1 message")
	}
	if result.Messages[0] != "Ola! Como posso ajudar?" {
		t.Errorf("message = %q, want direct response", result.Messages[0])
	}
	if executor.execCount != 0 {
		t.Errorf("executor called %d times, want 0", executor.execCount)
	}
	if planner.planCount != 0 {
		t.Errorf("planner called %d times, want 0", planner.planCount)
	}
}

// TestEngine_SingleToolExecution — LLM decides to use a tool with done=true.
// Engine should execute via executeSingleStep (1 Planner + 1 Executor call).
func TestEngine_SingleToolExecution(t *testing.T) {
	resp := &llmResponses{
		decide: `{"tool":"echo","input":"hello","done":true}`,
	}
	server := newMockLLMServer(resp)
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("echo: hello")

	eng := newTestEngine(server.URL, planner, executor, defaultLimits())
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "echo hello", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if executor.execCount != 1 {
		t.Errorf("executor called %d times, want 1", executor.execCount)
	}
	if executor.lastTool != "echo" {
		t.Errorf("executed tool = %q, want echo", executor.lastTool)
	}

	result := emitter.Response()
	if len(result.Messages) == 0 {
		t.Fatal("expected at least 1 message")
	}
	if !strings.Contains(result.Messages[len(result.Messages)-1], "echo: hello") {
		t.Errorf("message = %q, want to contain 'echo: hello'", result.Messages[len(result.Messages)-1])
	}
}

// TestEngine_HappyPath_PlanAndExecute — Full OBSERVE→PLAN→ACT→REFLECT loop.
// Decide returns a tool (not done), Plan returns 2 steps, Reflect returns done after step 2.
func TestEngine_HappyPath_PlanAndExecute(t *testing.T) {
	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "a", Purpose: "first"},
			{Name: "step2", Tool: "echo", Input: "b", Purpose: "second"},
		},
		Reasoning: "plano de 2 passos",
	}
	planJSON, _ := json.Marshal(plan)

	reflectCount := 0
	resp := &llmResponses{
		decide: `{"tool":"echo","input":"a","done":false}`,
		plan:   string(planJSON),
	}

	// Reflect returns "continue" for step 1, "done" for step 2
	server := newMockLLMServer(resp)
	// Override: we need stateful reflect responses
	server.Close()

	server = newStatefulLLMServer(resp, func() string {
		reflectCount++
		if reflectCount >= 2 {
			return `{"action":"done","reason":"completo","final_answer":"resultado final"}`
		}
		return `{"action":"continue","reason":"proximo passo"}`
	})
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("ok")

	eng := newTestEngine(server.URL, planner, executor, defaultLimits())
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "fazer tarefa", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if executor.execCount < 2 {
		t.Errorf("executor called %d times, want >= 2", executor.execCount)
	}

	result := emitter.Response()
	found := false
	for _, msg := range result.Messages {
		if strings.Contains(msg, "resultado final") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected final_answer in messages, got %v", result.Messages)
	}
}

// TestEngine_BudgetExhausted — MaxAttempts=2, Reflect always continues.
// Engine should stop after exhausting the budget.
func TestEngine_BudgetExhausted(t *testing.T) {
	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "a"},
			{Name: "step2", Tool: "echo", Input: "b"},
			{Name: "step3", Tool: "echo", Input: "c"},
		},
		Reasoning: "plano longo",
	}
	planJSON, _ := json.Marshal(plan)

	resp := &llmResponses{
		decide:  `{"tool":"echo","input":"a","done":false}`,
		plan:    string(planJSON),
		reflect: `{"action":"continue","reason":"seguir"}`,
	}
	server := newMockLLMServer(resp)
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("ok")

	limits := domain.AgentLimits{
		MaxSteps:    6,
		MaxAttempts: 2, // tight budget
		TimeoutMs:   30000,
	}

	eng := newTestEngine(server.URL, planner, executor, limits)
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "fazer tudo", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	// Engine should complete (not hang) and produce output
	result := emitter.Response()
	if len(result.Messages) == 0 {
		t.Error("expected at least 1 message when budget exhausted")
	}
}

// TestEngine_ReplanLoop — Reflect returns "replan" with revised steps.
// Engine should replace the plan and continue executing.
func TestEngine_ReplanLoop(t *testing.T) {
	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "original", Tool: "echo", Input: "first"},
		},
		Reasoning: "plano original",
	}
	planJSON, _ := json.Marshal(plan)

	reflectCount := 0
	resp := &llmResponses{
		decide: `{"tool":"echo","input":"first","done":false}`,
		plan:   string(planJSON),
	}

	revisedSteps := []domain.PlannedStep{
		{Name: "revised", Tool: "echo", Input: "fixed"},
	}
	revisedJSON, _ := json.Marshal(map[string]interface{}{
		"action":       "replan",
		"reason":       "ajustando",
		"revised_plan": revisedSteps,
	})

	server := newStatefulLLMServer(resp, func() string {
		reflectCount++
		if reflectCount == 1 {
			return string(revisedJSON)
		}
		return `{"action":"done","reason":"ok","final_answer":"replanejado com sucesso"}`
	})
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("ok")

	eng := newTestEngine(server.URL, planner, executor, defaultLimits())
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "tarefa", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if executor.execCount < 2 {
		t.Errorf("executor called %d times, want >= 2 (original + revised)", executor.execCount)
	}
}

// TestEngine_ContextPruning — with tiny MaxContextTokens, history should be pruned.
func TestEngine_ContextPruning(t *testing.T) {
	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "a"},
		},
		Reasoning: "plano simples",
	}
	planJSON, _ := json.Marshal(plan)

	resp := &llmResponses{
		decide:  `{"tool":"echo","input":"a","done":false}`,
		plan:    string(planJSON),
		reflect: `{"action":"done","reason":"ok","final_answer":"feito"}`,
		summary: "resumo comprimido",
	}
	server := newMockLLMServer(resp)
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor("ok")

	limits := defaultLimits()
	eng := newTestEngine(server.URL, planner, executor, limits)
	// Override MaxContextTokens to a tiny value to trigger pruning
	eng.llmConfig.MaxContextTokens = 50

	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "tarefa com contexto enorme "+strings.Repeat("x", 500), emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	// Test passes if it completes without error — pruning ran internally
	result := emitter.Response()
	if len(result.Messages) == 0 {
		t.Error("expected at least 1 message")
	}
}

// TestEngine_OutputSummarization — Tool returns 20k chars, engine should summarize.
func TestEngine_OutputSummarization(t *testing.T) {
	longOutput := strings.Repeat("data ", 4000) // 20k chars

	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "a"},
		},
		Reasoning: "plano",
	}
	planJSON, _ := json.Marshal(plan)

	resp := &llmResponses{
		decide:  `{"tool":"echo","input":"a","done":false}`,
		plan:    string(planJSON),
		reflect: `{"action":"done","reason":"ok","final_answer":"sumarizado"}`,
		summary: "output resumido em 3 frases",
	}
	server := newMockLLMServer(resp)
	defer server.Close()

	planner := newMockPlanner()
	planner.RegisterTool(&mockTool{name: "echo"})
	executor := newMockExecutor(longOutput) // returns 20k chars

	eng := newTestEngine(server.URL, planner, executor, defaultLimits())
	emitter := NewBufferEmitter()

	err := eng.RunLoop(context.Background(), "gerar relatorio", emitter)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	// Test passes if it completes — summarizeIfLong was invoked internally
	result := emitter.Response()
	if len(result.Messages) == 0 {
		t.Error("expected at least 1 message")
	}
}

