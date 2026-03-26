package agent

import (
	"database/sql"
	"testing"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	_ "github.com/glebarez/go-sqlite"
	"go.uber.org/zap"
)

func setupTestMemory(t *testing.T) (*SQLiteMemory, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewSQLiteMemory(db, zap.NewNop()), db
}

func TestSQLiteMemory_Save(t *testing.T) {
	mem, _ := setupTestMemory(t)
	err := mem.Save(domain.MemoryEntry{Content: "always use English", Category: "rule"})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	rules, err := mem.SearchByCategory("", "rule", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Content != "always use English" {
		t.Fatalf("expected 'always use English', got %q", rules[0].Content)
	}
}

func TestSQLiteMemory_DeleteRulesByContent(t *testing.T) {
	mem, _ := setupTestMemory(t)
	mem.Save(domain.MemoryEntry{Content: "always use tabs", Category: "rule"})
	mem.Save(domain.MemoryEntry{Content: "never use globals", Category: "rule"})

	deleted, err := mem.DeleteRulesByContent("tabs")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected 1 remaining rule, got %d", len(rules))
	}
	if rules[0].Content != "never use globals" {
		t.Fatalf("wrong rule survived: %q", rules[0].Content)
	}
}

func TestSQLiteMemory_DeleteRulesNoMatch(t *testing.T) {
	mem, _ := setupTestMemory(t)
	mem.Save(domain.MemoryEntry{Content: "always use tabs", Category: "rule"})

	deleted, err := mem.DeleteRulesByContent("nonexistent_keyword")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected 0 deleted, got %d", deleted)
	}

	rules, _ := mem.SearchByCategory("", "rule", 10)
	if len(rules) != 1 {
		t.Fatalf("expected rule to survive, got %d", len(rules))
	}
}

func TestSQLiteMemory_DeleteRulesCaseInsensitive(t *testing.T) {
	mem, _ := setupTestMemory(t)
	mem.Save(domain.MemoryEntry{Content: "Always use UPPERCASE", Category: "rule"})

	deleted, err := mem.DeleteRulesByContent("uppercase")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted (case-insensitive), got %d", deleted)
	}
}

func TestSQLiteMemory_SearchByCategory(t *testing.T) {
	mem, _ := setupTestMemory(t)
	mem.Save(domain.MemoryEntry{Content: "rule one", Category: "rule"})
	mem.Save(domain.MemoryEntry{Content: "rule two", Category: "rule"})
	mem.Save(domain.MemoryEntry{Content: "some fact", Category: "fact"})

	rules, err := mem.SearchByCategory("", "rule", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	facts, _ := mem.SearchByCategory("", "fact", 10)
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
}
