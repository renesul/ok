package integration

import (
	"context"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/repository"
	"go.uber.org/zap"
)

// --- AuditLog ---

func TestAuditLogRecord(t *testing.T) {
	defer cleanupAudit(t)

	audit := agent.NewAuditLog(testDB, zap.NewNop())
	audit.Record(agent.AuditEntry{
		Tool:     "echo",
		Input:    "hello",
		Output:   "hello",
		Safety:   "safe",
		Approved: true,
	})

	// Verify record exists
	var count int64
	testDB.QueryRow("SELECT COUNT(*) FROM agent_audit WHERE tool = 'echo'").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 audit record, got %d", count)
	}
}

func TestAuditLogRecordAutoID(t *testing.T) {
	defer cleanupAudit(t)

	audit := agent.NewAuditLog(testDB, zap.NewNop())
	audit.Record(agent.AuditEntry{
		Tool:   "http",
		Input:  "https://example.com",
		Output: "html content",
		Safety: "restricted",
	})

	var id string
	testDB.QueryRow("SELECT id FROM agent_audit WHERE tool = 'http'").Scan(&id)
	if id == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestAuditLogTruncatesOutput(t *testing.T) {
	defer cleanupAudit(t)

	audit := agent.NewAuditLog(testDB, zap.NewNop())
	longOutput := ""
	for i := 0; i < 200; i++ {
		longOutput += "abcdefghij" // 2000 chars
	}

	audit.Record(agent.AuditEntry{
		Tool:   "test",
		Input:  "test",
		Output: longOutput,
		Safety: "safe",
	})

	var storedOutput string
	testDB.QueryRow("SELECT output FROM agent_audit WHERE tool = 'test'").Scan(&storedOutput)
	if len(storedOutput) > 510 { // 500 + ellipsis
		t.Errorf("expected truncated output (<= 510 chars), got %d chars", len(storedOutput))
	}
}

// --- ConfigRepository ---

func TestConfigRepositoryGetSet(t *testing.T) {
	ctx := context.Background()

	err := testConfigRepo.Set(ctx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("set config failed: %v", err)
	}

	value, err := testConfigRepo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if value != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", value)
	}

	// Clean up
	testDB.Exec("DELETE FROM agent_config WHERE key = 'test_key'")
}

func TestConfigRepositoryOverwrite(t *testing.T) {
	ctx := context.Background()

	testConfigRepo.Set(ctx, "overwrite_key", "first")
	testConfigRepo.Set(ctx, "overwrite_key", "second")

	value, _ := testConfigRepo.Get(ctx, "overwrite_key")
	if value != "second" {
		t.Errorf("expected 'second' after overwrite, got '%s'", value)
	}

	testDB.Exec("DELETE FROM agent_config WHERE key = 'overwrite_key'")
}

func TestConfigRepositoryGetNonexistent(t *testing.T) {
	ctx := context.Background()

	value, err := testConfigRepo.Get(ctx, "totally_nonexistent_key_12345")
	if err != nil {
		t.Fatalf("get should not error for missing key: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty string for missing key, got '%s'", value)
	}
}

// --- ExecutionRepository ---

func TestExecutionRepositorySaveAndFind(t *testing.T) {
	defer cleanupExecutions(t)

	record := &domain.ExecutionRecord{
		ID:            "exec-test-1",
		Goal:          "buscar dados",
		Status:        "done",
		TotalMs:       1234,
		StepCount:     3,
		Steps:         []domain.StepResult{{Name: "step1", Tool: "echo", Status: "done"}},
		Timeline:      []domain.ExecutionEntry{{Phase: "observe", Content: "inicio"}},
		ToolsUsed:     []string{"echo", "http"},
		FailureReason: "",
		CreatedAt:     time.Now(),
	}

	err := testExecRepo.Save(record)
	if err != nil {
		t.Fatalf("save execution failed: %v", err)
	}

	found, err := testExecRepo.FindByID("exec-test-1")
	if err != nil {
		t.Fatalf("find execution failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected execution record, got nil")
	}
	if found.Goal != "buscar dados" {
		t.Errorf("expected goal 'buscar dados', got '%s'", found.Goal)
	}
	if found.Status != "done" {
		t.Errorf("expected status 'done', got '%s'", found.Status)
	}
	if found.TotalMs != 1234 {
		t.Errorf("expected total_ms 1234, got %d", found.TotalMs)
	}
	if found.StepCount != 3 {
		t.Errorf("expected step_count 3, got %d", found.StepCount)
	}
	if len(found.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(found.Steps))
	}
	if len(found.ToolsUsed) != 2 {
		t.Errorf("expected 2 tools used, got %d", len(found.ToolsUsed))
	}
}

func TestExecutionRepositoryFindByIDNotFound(t *testing.T) {
	defer cleanupExecutions(t)

	found, err := testExecRepo.FindByID("nonexistent-exec-id")
	if err != nil {
		t.Fatalf("find should not error: %v", err)
	}
	if found != nil {
		t.Error("expected nil for nonexistent execution")
	}
}

func TestExecutionRepositoryFindRecent(t *testing.T) {
	defer cleanupExecutions(t)

	for i := 0; i < 3; i++ {
		testExecRepo.Save(&domain.ExecutionRecord{
			Goal:      "goal " + itoa(i),
			Status:    "done",
			CreatedAt: time.Now(),
		})
	}

	records, err := testExecRepo.FindRecent(10)
	if err != nil {
		t.Fatalf("find recent failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	// Should be ordered by created_at DESC (most recent first)
	if records[0].Goal != "goal 2" {
		t.Errorf("expected most recent goal first, got '%s'", records[0].Goal)
	}
}

func TestExecutionRepositoryGetMetrics(t *testing.T) {
	defer cleanupExecutions(t)

	testExecRepo.Save(&domain.ExecutionRecord{
		Goal: "m1", Status: "done", TotalMs: 100, StepCount: 2,
		ToolsUsed: []string{"echo"}, CreatedAt: time.Now(),
	})
	testExecRepo.Save(&domain.ExecutionRecord{
		Goal: "m2", Status: "failed", TotalMs: 200, StepCount: 1,
		ToolsUsed: []string{"http"}, CreatedAt: time.Now(),
	})

	metrics, err := testExecRepo.GetMetrics()
	if err != nil {
		t.Fatalf("get metrics failed: %v", err)
	}
	if metrics.TotalExecutions != 2 {
		t.Errorf("expected 2 total executions, got %d", metrics.TotalExecutions)
	}
	if metrics.SuccessRate < 0.4 || metrics.SuccessRate > 0.6 {
		t.Errorf("expected ~0.5 success rate, got %f", metrics.SuccessRate)
	}
	if metrics.ToolUsageCount["echo"] != 1 {
		t.Errorf("expected echo usage 1, got %d", metrics.ToolUsageCount["echo"])
	}
	if metrics.ToolUsageCount["http"] != 1 {
		t.Errorf("expected http usage 1, got %d", metrics.ToolUsageCount["http"])
	}
}

// --- Session Expiration ---

func TestSessionExpiration(t *testing.T) {
	defer cleanupSessions(t)

	sessionRepo := repository.NewSessionRepository(testDB, zap.NewNop())
	ctx := context.Background()

	// Create expired session
	expired := &domain.Session{
		ID:        "expired-session-123",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-8 * time.Hour),
	}
	err := sessionRepo.Create(ctx, expired)
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	// Create valid session
	valid := &domain.Session{
		ID:        "valid-session-456",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, valid)
	if err != nil {
		t.Fatalf("create valid session: %v", err)
	}

	// Delete expired
	err = sessionRepo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("delete expired: %v", err)
	}

	// Expired should be gone
	found, _ := sessionRepo.FindByID(ctx, "expired-session-123")
	if found != nil {
		t.Error("expired session should have been deleted")
	}

	// Valid should remain
	found, _ = sessionRepo.FindByID(ctx, "valid-session-456")
	if found == nil {
		t.Error("valid session should still exist")
	}
}

// --- Memory SearchByCategory ---

func TestMemorySearchByCategory(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	memory.Save(domain.MemoryEntry{Content: "regra: sempre portugues", Category: "rule"})
	memory.Save(domain.MemoryEntry{Content: "fato: usuario gosta de Go", Category: "fact"})
	memory.Save(domain.MemoryEntry{Content: "regra: nunca iframes", Category: "rule"})

	rules, err := memory.SearchByCategory("", "rule", 10)
	if err != nil {
		t.Fatalf("search by category failed: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	facts, _ := memory.SearchByCategory("", "fact", 10)
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
}

// --- Memory SaveChunked ---

func TestMemorySaveChunked(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())

	// Create content larger than maxContentLength (4000)
	longContent := ""
	for i := 0; i < 600; i++ {
		longContent += "palavra " // ~4800 chars
	}

	err := memory.SaveChunked(domain.MemoryEntry{Content: longContent, Category: "fact"})
	if err != nil {
		t.Fatalf("save chunked failed: %v", err)
	}

	// Should have saved multiple chunks
	all, _ := memory.Recent(20)
	if len(all) < 2 {
		t.Errorf("expected multiple chunks, got %d entries", len(all))
	}
}

// --- Memory Recent ---

func TestMemoryRecent(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	memory.Save(domain.MemoryEntry{Content: "first"})
	time.Sleep(10 * time.Millisecond)
	memory.Save(domain.MemoryEntry{Content: "second"})
	time.Sleep(10 * time.Millisecond)
	memory.Save(domain.MemoryEntry{Content: "third"})

	recent, err := memory.Recent(2)
	if err != nil {
		t.Fatalf("recent failed: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2, got %d", len(recent))
	}
	if recent[0].Content != "third" {
		t.Errorf("expected 'third' first (most recent), got '%s'", recent[0].Content)
	}
}
