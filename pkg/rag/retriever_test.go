package rag

import (
	"context"
	"testing"
	"time"
)

// mockEmbedder returns pre-determined embeddings for testing.
type mockEmbedder struct {
	embeddings map[string][]float32
	model      string
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if emb, ok := m.embeddings[text]; ok {
		return emb, nil
	}
	// Default: return a zero vector
	return make([]float32, 4), nil
}

func (m *mockEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, t := range texts {
		emb, _ := m.Embed(nil, t)
		result[i] = emb
	}
	return result, nil
}

func (m *mockEmbedder) Model() string { return m.model }

func TestRetriever_IndexAndSearch(t *testing.T) {
	store := NewInteractionStore(t.TempDir())
	store.Open()

	embedder := &mockEmbedder{
		model: "test-model",
		embeddings: map[string][]float32{
			"sunny weather":     {1, 0, 0, 0},
			"go programming":    {0, 1, 0, 0},
			"weather forecast":  {0.9, 0.1, 0, 0},
		},
	}

	retriever := NewRetriever(store, embedder, 3, 0.5)

	// Index interactions
	ctx := context.Background()
	retriever.Index(ctx, Interaction{
		ID: "1", Role: "user", Content: "sunny weather", Timestamp: time.Now(),
	})
	retriever.Index(ctx, Interaction{
		ID: "2", Role: "user", Content: "go programming", Timestamp: time.Now(),
	})

	// Search
	results, err := retriever.Search(ctx, "weather forecast")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (above 0.5 threshold), got %d", len(results))
	}
	if results[0].Interaction.ID != "1" {
		t.Errorf("expected ID '1', got %q", results[0].Interaction.ID)
	}
}

func TestRetriever_EmptyContent(t *testing.T) {
	store := NewInteractionStore(t.TempDir())
	store.Open()

	embedder := &mockEmbedder{model: "test"}
	retriever := NewRetriever(store, embedder, 5, 0.5)

	// Empty content should be skipped
	err := retriever.Index(context.Background(), Interaction{
		ID: "1", Role: "user", Content: "  ", Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.Count() != 0 {
		t.Fatalf("expected 0 (empty content skipped), got %d", store.Count())
	}
}

func TestRetriever_SearchEmptyStore(t *testing.T) {
	store := NewInteractionStore(t.TempDir())
	store.Open()

	embedder := &mockEmbedder{model: "test"}
	retriever := NewRetriever(store, embedder, 5, 0.5)

	results, err := retriever.Search(context.Background(), "anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestFormatContext(t *testing.T) {
	results := []ScoredInteraction{
		{
			Interaction: Interaction{
				Role:      "user",
				Content:   "What is Go?",
				Timestamp: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			Score: 0.9,
		},
	}

	output := FormatContext(results)
	if output == "" {
		t.Fatal("expected non-empty output")
	}

	expected := "## Relevant Past Interactions"
	if len(output) < len(expected) || output[:len(expected)] != expected {
		t.Errorf("unexpected format: %q", output[:50])
	}
}

func TestFormatContext_Empty(t *testing.T) {
	if got := FormatContext(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
