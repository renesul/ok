package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

func newTestContextManager(maxTokens int) *ContextManager {
	return NewContextManager(
		llm.ClientConfig{MaxContextTokens: maxTokens},
		func() string { return "system prompt" },
		zap.NewNop(),
	)
}

func stateWithEntries(n int) *domain.ExecutionState {
	state := agentpkg.NewExecutionState("test", domain.ExecutionBudget{
		MaxSteps:    10,
		MaxDuration: 30 * time.Second,
	})
	for i := 0; i < n; i++ {
		agentpkg.AddEntry(state, domain.PhaseObserve, strings.Repeat("x", 100))
	}
	return state
}

func TestPruneContext_NoPruneUnderThreshold(t *testing.T) {
	cm := newTestContextManager(128000)
	state := stateWithEntries(5)
	before := len(state.History)
	cm.PruneContextIfNeeded(context.Background(), state)
	if len(state.History) != before {
		t.Fatalf("expected %d entries, got %d", before, len(state.History))
	}
}

func TestPruneContext_PrunesOver80Percent(t *testing.T) {
	cm := newTestContextManager(50) // very small window
	state := stateWithEntries(10)
	cm.PruneContextIfNeeded(context.Background(), state)
	if len(state.History) >= 10 {
		t.Fatalf("expected pruned history, got %d entries", len(state.History))
	}
	if !strings.Contains(state.History[0].Content, "podado") {
		t.Fatalf("expected first entry to contain 'podado', got %q", state.History[0].Content)
	}
}

func TestPruneContext_SkipsFewerThan4(t *testing.T) {
	cm := newTestContextManager(1) // forces over threshold
	state := stateWithEntries(3)
	cm.PruneContextIfNeeded(context.Background(), state)
	if len(state.History) != 3 {
		t.Fatalf("expected 3 entries (no pruning), got %d", len(state.History))
	}
}

func TestPruneContext_ZeroMaxTokens(t *testing.T) {
	cm := newTestContextManager(0)
	state := stateWithEntries(10)
	before := len(state.History)
	cm.PruneContextIfNeeded(context.Background(), state)
	if len(state.History) != before {
		t.Fatalf("expected %d entries with zero max tokens, got %d", before, len(state.History))
	}
}

func TestSummarizeIfLong_Short(t *testing.T) {
	cm := newTestContextManager(128000)
	input := strings.Repeat("a", 100)
	result := cm.SummarizeIfLong(context.Background(), input)
	if result != input {
		t.Fatalf("short output should be returned as-is")
	}
}

func TestSummarizeIfLong_Long(t *testing.T) {
	cm := newTestContextManager(128000)
	input := strings.Repeat("a", 5000)
	result := cm.SummarizeIfLong(context.Background(), input)
	if len(result) > 1503 { // 1500 + "..."
		t.Fatalf("expected truncated output <= 1503, got %d", len(result))
	}
}

func TestSummarizeIfLong_ExactBoundary(t *testing.T) {
	cm := newTestContextManager(128000)
	input := strings.Repeat("a", 1500)
	result := cm.SummarizeIfLong(context.Background(), input)
	if result != input {
		t.Fatalf("boundary output (1500) should be returned as-is")
	}
}

func TestEstimateTokens(t *testing.T) {
	cm := newTestContextManager(128000)
	cases := []struct {
		input    string
		expected int
	}{
		{"1234", 1},
		{"12345678", 2},
		{"", 0},
		{strings.Repeat("x", 400), 100},
	}
	for _, c := range cases {
		got := cm.estimateTokens(c.input)
		if got != c.expected {
			t.Errorf("estimateTokens(%d chars) = %d, want %d", len(c.input), got, c.expected)
		}
	}
}
