package integration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	"go.uber.org/zap"
)

// Tool tests

func TestEchoTool(t *testing.T) {
	tool := &agenttools.EchoTool{}

	if tool.Name() != "echo" {
		t.Errorf("expected name 'echo', got '%s'", tool.Name())
	}

	result, err := tool.Run("hello world")
	if err != nil {
		t.Fatalf("echo tool error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result)
	}
}

func TestHTTPGetTool(t *testing.T) {
	tool := agenttools.NewHTTPTool()

	if tool.Name() != "http" {
		t.Errorf("expected name 'http', got '%s'", tool.Name())
	}

	_, err := tool.Run("")
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

// Context tests

func TestAgentContext(t *testing.T) {
	ctx := domain.NewAgentContext(5)
	ctx.Add("user: hello")
	ctx.Add("tool:echo result: hello")

	expected := "user: hello\ntool:echo result: hello"
	if ctx.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, ctx.String())
	}
}

func TestAgentContextLimitReached(t *testing.T) {
	ctx := domain.NewAgentContext(2)
	if ctx.LimitReached() {
		t.Error("should not be reached at 0 steps")
	}

	ctx.Increment()
	ctx.Increment()

	if !ctx.LimitReached() {
		t.Error("should be reached at 2 steps with max 2")
	}
}

// Planner tests

func TestPlannerValidTool(t *testing.T) {
	log := zap.NewNop()
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(&agenttools.EchoTool{})

	ctx := domain.NewAgentContext(5)
	decision := domain.Decision{Tool: "echo", Input: "hello", Done: false}

	plan, err := planner.Plan(decision, ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if plan.Tool.Name() != "echo" {
		t.Errorf("expected tool 'echo', got '%s'", plan.Tool.Name())
	}
	if plan.Input != "hello" {
		t.Errorf("expected input 'hello', got '%s'", plan.Input)
	}
}

func TestPlannerInvalidTool(t *testing.T) {
	log := zap.NewNop()
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(&agenttools.EchoTool{})

	ctx := domain.NewAgentContext(5)
	decision := domain.Decision{Tool: "nonexistent", Input: "test", Done: false}

	_, err := planner.Plan(decision, ctx)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestPlannerEmptyInput(t *testing.T) {
	log := zap.NewNop()
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(agenttools.NewHTTPTool())

	ctx := domain.NewAgentContext(5)
	decision := domain.Decision{Tool: "http", Input: "", Done: false}

	_, err := planner.Plan(decision, ctx)
	if err == nil {
		t.Error("expected error for empty input on http tool")
	}
}

func TestPlannerEmptyInputEchoAllowed(t *testing.T) {
	log := zap.NewNop()
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(&agenttools.EchoTool{})

	ctx := domain.NewAgentContext(5)
	decision := domain.Decision{Tool: "echo", Input: "", Done: false}

	_, err := planner.Plan(decision, ctx)
	if err != nil {
		t.Fatalf("echo with empty input should be allowed, got: %v", err)
	}
}

func TestPlannerLimitReached(t *testing.T) {
	log := zap.NewNop()
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(&agenttools.EchoTool{})

	ctx := domain.NewAgentContext(2)
	ctx.Increment()
	ctx.Increment()

	decision := domain.Decision{Tool: "echo", Input: "test", Done: false}

	_, err := planner.Plan(decision, ctx)
	if err == nil {
		t.Error("expected error when limit reached")
	}
}

// Executor tests

func TestExecutorSuccess(t *testing.T) {
	log := zap.NewNop()
	executor := agent.NewDefaultExecutor(log)

	plan := domain.Plan{
		Tool:  &agenttools.EchoTool{},
		Input: "hello",
	}

	result, err := executor.Execute(plan)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestExecutorError(t *testing.T) {
	log := zap.NewNop()
	executor := agent.NewDefaultExecutor(log)

	plan := domain.Plan{
		Tool:  agenttools.NewHTTPTool(),
		Input: "not-a-valid-url",
	}

	_, err := executor.Execute(plan)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

// Memory tests

func TestMemorySave(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())

	err := memory.Save(domain.MemoryEntry{Content: "google: https://google.com -> pagina inicial"})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	results, err := memory.Search("google", 5)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "google: https://google.com -> pagina inicial" {
		t.Errorf("unexpected content: %s", results[0].Content)
	}
}

func TestMemorySearchMultiple(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	memory.Save(domain.MemoryEntry{Content: "http: google.com -> html"})
	memory.Save(domain.MemoryEntry{Content: "http: github.com -> html"})
	memory.Save(domain.MemoryEntry{Content: "echo: hello -> hello"})

	results, _ := memory.Search("http", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'http', got %d", len(results))
	}

	results, _ = memory.Search("hello", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'hello', got %d", len(results))
	}
}

func TestMemorySearchEmpty(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	results, err := memory.Search("nonexistent", 5)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestShouldStoreValid(t *testing.T) {
	if !agent.ShouldStore("google.com", "pagina html com conteudo") {
		t.Error("expected true for valid input/output")
	}
}

func TestShouldStoreEmpty(t *testing.T) {
	if agent.ShouldStore("test", "") {
		t.Error("expected false for empty output")
	}
}

func TestShouldStoreError(t *testing.T) {
	if agent.ShouldStore("test", "execution error: timeout") {
		t.Error("expected false for error output")
	}
}

func TestShouldStoreShort(t *testing.T) {
	if agent.ShouldStore("a", "b") {
		t.Error("expected false for short input+output")
	}
}

// HTTP Endpoint tests

func TestAgentEndpointRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "POST", "/api/agent/run")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAgentEndpointEmptyInput(t *testing.T) {
	defer cleanupAll(t)

	body := `{"input":""}`
	resp := authenticatedRequest(t, "POST", "/api/agent/run", bytes.NewBufferString(body))
	if resp.StatusCode != 422 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestAgentStreamRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "POST", "/api/agent/stream")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAgentStreamEmptyInput(t *testing.T) {
	defer cleanupAll(t)

	body := `{"input":""}`
	resp := authenticatedRequest(t, "POST", "/api/agent/stream", bytes.NewBufferString(body))
	if resp.StatusCode != 422 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// System Tools tests

func TestFileReadTool(t *testing.T) {
	sandboxDir := setupSandbox(t)
	defer os.RemoveAll(sandboxDir)

	os.WriteFile(sandboxDir+"/test.txt", []byte("hello world"), 0644)

	tool := agenttools.NewFileReadTool(sandboxDir)
	result, err := tool.Run(`{"file":"test.txt"}`)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("expected result to contain 'hello world', got '%s'", result)
	}
}

func TestFileReadToolFallbackInput(t *testing.T) {
	sandboxDir := setupSandbox(t)
	defer os.RemoveAll(sandboxDir)

	os.WriteFile(sandboxDir+"/test.txt", []byte("hello world"), 0644)

	tool := agenttools.NewFileReadTool(sandboxDir)
	result, err := tool.Run("test.txt")
	if err != nil {
		t.Fatalf("fallback read failed: %v", err)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("expected result to contain 'hello world', got '%s'", result)
	}
}

func TestFileReadToolPathTraversal(t *testing.T) {
	tool := agenttools.NewFileReadTool("/tmp/sandbox")
	_, err := tool.Run(`{"file":"../../../etc/passwd"}`)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestFileReadToolBinaryRejected(t *testing.T) {
	sandboxDir := setupSandbox(t)
	defer os.RemoveAll(sandboxDir)

	os.WriteFile(sandboxDir+"/data.sqlite", []byte("SQLite format 3\x00"), 0644)

	tool := agenttools.NewFileReadTool(sandboxDir)
	_, err := tool.Run(`{"file":"data.sqlite"}`)
	if err == nil {
		t.Error("expected error for binary file")
	}
}

func TestFileReadToolPagination(t *testing.T) {
	sandboxDir := setupSandbox(t)
	defer os.RemoveAll(sandboxDir)

	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d content", i))
	}
	os.WriteFile(sandboxDir+"/big.txt", []byte(strings.Join(lines, "\n")), 0644)

	tool := agenttools.NewFileReadTool(sandboxDir)
	result, err := tool.Run(`{"file":"big.txt","start_line":50,"end_line":60}`)
	if err != nil {
		t.Fatalf("paginated read failed: %v", err)
	}
	if !strings.Contains(result, "line 50") {
		t.Error("expected result to contain line 50")
	}
	if !strings.Contains(result, "line 60") {
		t.Error("expected result to contain line 60")
	}
	if strings.Contains(result, "line 49") {
		t.Error("should not contain line 49")
	}
}

func TestFileWriteTool(t *testing.T) {
	sandboxDir := setupSandbox(t)
	defer os.RemoveAll(sandboxDir)

	tool := agenttools.NewFileWriteTool(sandboxDir)
	input := `{"path":"output.txt","content":"test content"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if result != "arquivo escrito: output.txt" {
		t.Errorf("unexpected result: %s", result)
	}

	data, _ := os.ReadFile(sandboxDir + "/output.txt")
	if string(data) != "test content" {
		t.Errorf("file content mismatch: %s", string(data))
	}
}

func TestFileWriteToolPathTraversal(t *testing.T) {
	tool := agenttools.NewFileWriteTool("/tmp/sandbox")
	input := `{"path":"../../etc/evil","content":"bad"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestShellToolAllowed(t *testing.T) {
	tool := agenttools.NewShellTool()
	result, err := tool.Run("echo hello")
	if err != nil {
		t.Fatalf("shell failed: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("expected result to contain 'hello', got '%s'", result)
	}
}

func TestShellToolBlocked(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("rm -rf /")
	if err == nil {
		t.Error("expected error for blocked command")
	}
}

func TestShellToolPipesAllowed(t *testing.T) {
	tool := agenttools.NewShellTool()
	result, err := tool.Run("echo hello world | wc -w")
	if err != nil {
		t.Fatalf("pipe should work: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result from pipe")
	}
}

// HTTP Full Tool tests

func TestHTTPToolPost(t *testing.T) {
	tool := agenttools.NewHTTPTool()
	// POST to httpbin
	input := `{"method":"POST","url":"https://httpbin.org/post","body":"{\"key\":\"value\"}"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Skipf("skipping HTTP POST test (network): %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestHTTPToolSimpleURL(t *testing.T) {
	tool := agenttools.NewHTTPTool()
	// Simple string = GET
	result, err := tool.Run("https://httpbin.org/get")
	if err != nil {
		t.Skipf("skipping (network): %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// JSON Parse tests

func TestJSONParseTool(t *testing.T) {
	tool := &agenttools.JSONParseTool{}
	result, err := tool.Run(`{"json":"{\"name\":\"OK\",\"version\":1}","path":"name"}`)
	if err != nil {
		t.Fatalf("json parse error: %v", err)
	}
	if result != "OK" {
		t.Errorf("expected 'OK', got '%s'", result)
	}
}

func TestJSONParseToolNestedPath(t *testing.T) {
	tool := &agenttools.JSONParseTool{}
	result, err := tool.Run(`{"json":"{\"user\":{\"name\":\"Rene\"}}","path":"user.name"}`)
	if err != nil {
		t.Fatalf("json nested parse error: %v", err)
	}
	if result != "Rene" {
		t.Errorf("expected 'Rene', got '%s'", result)
	}
}

// Base64 tests

func TestBase64Encode(t *testing.T) {
	tool := &agenttools.Base64Tool{}
	result, err := tool.Run(`{"action":"encode","data":"hello"}`)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if result != "aGVsbG8=" {
		t.Errorf("expected 'aGVsbG8=', got '%s'", result)
	}
}

func TestBase64Decode(t *testing.T) {
	tool := &agenttools.Base64Tool{}
	result, err := tool.Run(`{"action":"decode","data":"aGVsbG8="}`)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

// Timestamp tests

func TestTimestampNow(t *testing.T) {
	tool := &agenttools.TimestampTool{}
	result, err := tool.Run("now")
	if err != nil {
		t.Fatalf("timestamp error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestTimestampUnix(t *testing.T) {
	tool := &agenttools.TimestampTool{}
	result, err := tool.Run("unix:1700000000")
	if err != nil {
		t.Fatalf("unix parse error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// Math tests

func TestMathSimple(t *testing.T) {
	tool := &agenttools.MathTool{}
	result, err := tool.Run("2+3*4")
	if err != nil {
		t.Fatalf("math error: %v", err)
	}
	if result != "14" {
		t.Errorf("expected '14', got '%s'", result)
	}
}

func TestMathParentheses(t *testing.T) {
	tool := &agenttools.MathTool{}
	result, err := tool.Run("(2+3)*4")
	if err != nil {
		t.Fatalf("math error: %v", err)
	}
	if result != "20" {
		t.Errorf("expected '20', got '%s'", result)
	}
}

func TestMathDivisionByZero(t *testing.T) {
	tool := &agenttools.MathTool{}
	_, err := tool.Run("10/0")
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

// Text Extract tests

func TestTextExtract(t *testing.T) {
	tool := &agenttools.TextExtractTool{}
	result, err := tool.Run("<html><body><h1>Hello</h1><p>World</p></body></html>")
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}
	if result != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", result)
	}
}

func TestTextExtractScript(t *testing.T) {
	tool := &agenttools.TextExtractTool{}
	result, err := tool.Run("<p>Text</p><script>alert(1)</script><p>More</p>")
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}
	if result != "Text More" {
		t.Errorf("expected 'Text More', got '%s'", result)
	}
}

func setupSandbox(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "agent-sandbox-*")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	return dir
}

// --- Autonomous Loop Tests ---

func newBudget(maxSteps int) domain.ExecutionBudget {
	return domain.ExecutionBudget{MaxSteps: maxSteps}
}

func TestExecutionStateCreation(t *testing.T) {
	state := agent.NewExecutionState("buscar dados", newBudget(12))

	if state.Goal != "buscar dados" {
		t.Errorf("expected goal 'buscar dados', got '%s'", state.Goal)
	}
	if state.Budget.MaxSteps != 12 {
		t.Errorf("expected max steps 12, got %d", state.Budget.MaxSteps)
	}
	if state.Phase != domain.PhaseObserve {
		t.Errorf("expected phase observe, got '%s'", state.Phase)
	}
	if state.CurrentStep != 0 {
		t.Errorf("expected current step 0, got %d", state.CurrentStep)
	}
	exhausted, _ := agent.IsBudgetExhausted(state)
	if exhausted {
		t.Error("new state should not be exhausted")
	}
}

func TestExecutionStateSetPlanAndAdvance(t *testing.T) {
	state := agent.NewExecutionState("objetivo", newBudget(12))

	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "hello", Purpose: "test"},
			{Name: "step2", Tool: "echo", Input: "world", Purpose: "test"},
		},
		Reasoning: "plano simples",
	}

	agent.SetPlan(state, plan)

	if len(state.Plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(state.Plan.Steps))
	}
	if state.Plan.Steps[0].Status != "pending" {
		t.Errorf("expected pending status, got '%s'", state.Plan.Steps[0].Status)
	}

	step := agent.CurrentPlannedStep(state)
	if step == nil || step.Name != "step1" {
		t.Error("expected current step to be step1")
	}

	agent.AdvanceStep(state)
	step = agent.CurrentPlannedStep(state)
	if step == nil || step.Name != "step2" {
		t.Error("expected current step to be step2")
	}

	agent.AdvanceStep(state)
	exhausted, reason := agent.IsBudgetExhausted(state)
	if !exhausted {
		t.Error("state should be exhausted after advancing past all steps")
	}
	if !containsSubstring(reason, "plano completo") {
		t.Errorf("expected 'plano completo' reason, got '%s'", reason)
	}
}

func TestExecutionStateReplan(t *testing.T) {
	state := agent.NewExecutionState("objetivo", newBudget(12))

	plan := domain.ExecutionPlan{
		Steps: []domain.PlannedStep{
			{Name: "step1", Tool: "echo", Input: "a"},
			{Name: "step2", Tool: "echo", Input: "b"},
			{Name: "step3", Tool: "echo", Input: "c"},
		},
	}
	agent.SetPlan(state, plan)

	state.Plan.Steps[0].Status = "done"
	state.Plan.Steps[0].Output = "resultado a"
	agent.AdvanceStep(state)

	newSteps := []domain.PlannedStep{
		{Name: "step_new", Tool: "http", Input: "url"},
	}
	agent.ReplacePlan(state, newSteps)

	if len(state.Plan.Steps) != 2 {
		t.Fatalf("expected 2 steps after replan (1 done + 1 new), got %d", len(state.Plan.Steps))
	}
	if state.Plan.Steps[0].Name != "step1" {
		t.Error("first step should still be step1 (completed)")
	}
	if state.Plan.Steps[1].Name != "step_new" {
		t.Errorf("second step should be step_new, got '%s'", state.Plan.Steps[1].Name)
	}
}

func TestExecutionStateBuildContext(t *testing.T) {
	state := agent.NewExecutionState("buscar info", newBudget(12))
	state.Memories = []string{"memoria anterior"}
	agent.AddEntry(state, domain.PhaseObserve, "user: buscar info")

	ctx := agent.BuildContext(state)

	if ctx == "" {
		t.Fatal("context should not be empty")
	}
	if !containsSubstring(ctx, "buscar info") {
		t.Error("context should contain goal")
	}
	if !containsSubstring(ctx, "memoria anterior") {
		t.Error("context should contain memories")
	}
}

func TestExecutionStateBudgetExhausted(t *testing.T) {
	state := agent.NewExecutionState("objetivo", newBudget(3))
	state.Attempts = 3

	exhausted, reason := agent.IsBudgetExhausted(state)
	if !exhausted {
		t.Error("should be exhausted when attempts >= max")
	}
	if !containsSubstring(reason, "max steps") {
		t.Errorf("expected 'max steps' reason, got '%s'", reason)
	}
}

func TestTransitionValid(t *testing.T) {
	state := agent.NewExecutionState("test", newBudget(12))

	if err := agent.Transition(state, domain.PhasePlan); err != nil {
		t.Errorf("observe -> plan should be valid, got: %v", err)
	}
	if state.Phase != domain.PhasePlan {
		t.Errorf("expected phase plan, got %s", state.Phase)
	}
}

func TestTransitionInvalid(t *testing.T) {
	state := agent.NewExecutionState("test", newBudget(12))

	if err := agent.Transition(state, domain.PhaseReflect); err == nil {
		t.Error("observe -> reflect should be invalid")
	}
}

func TestPlanningPromptFormat(t *testing.T) {
	prompt := agent.BuildPlanningPrompt("buscar dados", "echo: repete texto\nhttp: faz requisicao", []string{"memoria1"})

	if !containsSubstring(prompt, "buscar dados") {
		t.Error("prompt should contain goal")
	}
	if !containsSubstring(prompt, "echo") {
		t.Error("prompt should contain tool descriptions")
	}
	if !containsSubstring(prompt, "memoria1") {
		t.Error("prompt should contain memories")
	}
	if !containsSubstring(prompt, "JSON") {
		t.Error("prompt should instruct JSON response")
	}
}

func TestReflectionPromptFormat(t *testing.T) {
	steps := []domain.PlannedStep{
		{Name: "passo1", Tool: "echo", Status: "done", Output: "resultado"},
	}
	prompt := agent.BuildReflectionPrompt("objetivo", steps, "resultado")

	if !containsSubstring(prompt, "objetivo") {
		t.Error("prompt should contain goal")
	}
	if !containsSubstring(prompt, "passo1") {
		t.Error("prompt should contain step name")
	}
	if !containsSubstring(prompt, "resultado") {
		t.Error("prompt should contain last result")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsCheck(s, sub))
}

func containsCheck(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
