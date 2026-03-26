package application

import (
	"database/sql"
	"testing"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/llm"
	_ "github.com/glebarez/go-sqlite"
	"go.uber.org/zap"
)

func setupTestAgentService(t *testing.T) (*AgentService, *agent.SQLiteMemory) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mem := agent.NewSQLiteMemory(db, zap.NewNop())
	svc := &AgentService{
		memory: mem,
		log:    zap.NewNop(),
	}
	return svc, mem
}

func TestAutoLearn_FromNowOn(t *testing.T) {
	svc, mem := setupTestAgentService(t)
	svc.autoLearnRule("from now on always respond in English")

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule saved, got %d", len(rules))
	}
	if rules[0].Content != "from now on always respond in English" {
		t.Fatalf("wrong content: %q", rules[0].Content)
	}
}

func TestAutoLearn_Always(t *testing.T) {
	svc, mem := setupTestAgentService(t)
	svc.autoLearnRule("always use tabs instead of spaces")

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule saved, got %d", len(rules))
	}
}

func TestAutoLearn_NoMatch(t *testing.T) {
	svc, mem := setupTestAgentService(t)
	svc.autoLearnRule("what is the weather today")

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules (no pattern match), got %d", len(rules))
	}
}

func TestAutoLearn_NilMemory(t *testing.T) {
	svc := &AgentService{memory: nil, log: zap.NewNop()}
	// Should not panic
	svc.autoLearnRule("always do something")
}

func TestAutoForget_ForgetRule(t *testing.T) {
	svc, mem := setupTestAgentService(t)

	mem.Save(domain.MemoryEntry{Content: "always use tabs", Category: "rule"})
	mem.Save(domain.MemoryEntry{Content: "never use globals", Category: "rule"})

	forgot := svc.autoForgetRule("forget tabs")
	if !forgot {
		t.Fatal("expected autoForgetRule to return true")
	}

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule remaining, got %d", len(rules))
	}
	if rules[0].Content != "never use globals" {
		t.Fatalf("wrong rule survived: %q", rules[0].Content)
	}
}

func TestAutoForget_NoMatch(t *testing.T) {
	svc, _ := setupTestAgentService(t)
	forgot := svc.autoForgetRule("hello world")
	if forgot {
		t.Fatal("expected autoForgetRule to return false for non-matching input")
	}
}

func TestAutoForget_NilMemory(t *testing.T) {
	svc := &AgentService{memory: nil, log: zap.NewNop()}
	forgot := svc.autoForgetRule("forget everything")
	if forgot {
		t.Fatal("expected false with nil memory")
	}
}

func TestAutoLearn_Never(t *testing.T) {
	svc, mem := setupTestAgentService(t)
	svc.autoLearnRule("never commit automatically")

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for 'never' pattern, got %d", len(rules))
	}
}

func TestAutoLearn_RememberThis(t *testing.T) {
	svc, mem := setupTestAgentService(t)
	svc.autoLearnRule("remember this: my timezone is UTC-3")

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for 'remember this' pattern, got %d", len(rules))
	}
}

func TestAutoForgetPreventsAutoLearn(t *testing.T) {
	svc, mem := setupTestAgentService(t)

	mem.Save(domain.MemoryEntry{Content: "always use tabs", Category: "rule"})

	// "forget" should prevent "always" from triggering autoLearn
	input := "forget tabs"
	forgot := svc.autoForgetRule(input)
	if !forgot {
		svc.autoLearnRule(input)
	}

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules (forget should delete and prevent learn), got %d", len(rules))
	}
}

// --- ListTools / ListSkills Tests ---

type testSafeTool struct {
	name   string
	desc   string
	safety domain.ToolSafety
}

func (t *testSafeTool) Name() string                       { return t.name }
func (t *testSafeTool) Description() string                { return t.desc }
func (t *testSafeTool) Run(_ string) (string, error)       { return "", nil }
func (t *testSafeTool) Safety() domain.ToolSafety          { return t.safety }

func TestListTools_ReturnsRegisteredTools(t *testing.T) {
	svc := newTestAgentServiceFull(t)
	svc.planner.RegisterTool(&testSafeTool{name: "echo", desc: "echoes input", safety: "safe"})
	svc.planner.RegisterTool(&testSafeTool{name: "shell", desc: "runs commands", safety: "dangerous"})

	tools := svc.ListTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool["name"]] = true
	}
	if !names["echo"] || !names["shell"] {
		t.Fatalf("expected echo and shell, got %v", tools)
	}
}

func TestListTools_IncludesSafety(t *testing.T) {
	svc := newTestAgentServiceFull(t)
	svc.planner.RegisterTool(&testSafeTool{name: "shell", desc: "runs", safety: "dangerous"})

	tools := svc.ListTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0]["safety"] != "dangerous" {
		t.Fatalf("expected safety 'dangerous', got %q", tools[0]["safety"])
	}
}

func TestListSkills_NilRepo(t *testing.T) {
	svc := newTestAgentServiceFull(t)
	// skillRepo is nil by default in newTestAgentServiceFull
	skills := svc.ListSkills()
	if skills != nil {
		t.Fatalf("expected nil for nil skillRepo, got %v", skills)
	}
}

// newTestAgentServiceFull creates an AgentService with enough dependencies for Run()
func newTestAgentServiceFull(t *testing.T) *AgentService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mem := agent.NewSQLiteMemory(db, zap.NewNop())
	execRepo := agent.NewExecutionRepository(db, zap.NewNop())
	configRepo := agent.NewConfigRepository(db, zap.NewNop())
	planner := agent.NewDefaultPlanner(zap.NewNop())
	executor := agent.NewDefaultExecutor(zap.NewNop())

	return NewAgentService(
		db, llm.NewClient(zap.NewNop()),
		llm.ClientConfig{}, llm.ClientConfig{},
		planner, executor, mem, execRepo, configRepo, nil,
		zap.NewNop(),
	)
}
