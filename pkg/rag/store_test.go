package rag

import (
	"math"
	"os"
	"testing"
	"time"
)

func TestInteractionStore_AddAndSearch(t *testing.T) {
	dir := t.TempDir()
	store := NewInteractionStore(dir)
	if err := store.Open(); err != nil {
		t.Fatal(err)
	}

	// Add interactions with simple embeddings
	interactions := []struct {
		text string
		emb  []float32
	}{
		{"The weather is sunny today", []float32{1, 0, 0, 0}},
		{"I like programming in Go", []float32{0, 1, 0, 0}},
		{"The sun is shining bright", []float32{0.9, 0.1, 0, 0}},
	}

	for i, it := range interactions {
		err := store.Add(Interaction{
			ID:        string(rune('A' + i)),
			Role:      "user",
			Content:   it.text,
			Timestamp: time.Now(),
		}, it.emb)
		if err != nil {
			t.Fatal(err)
		}
	}

	if store.Count() != 3 {
		t.Fatalf("expected 3, got %d", store.Count())
	}

	// Search for something similar to weather ([1,0,0,0])
	results := store.Search([]float32{1, 0, 0, 0}, 2, 0.5)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be exact match "weather is sunny" (cosine=1.0)
	if results[0].Interaction.Content != "The weather is sunny today" {
		t.Errorf("expected 'The weather is sunny today', got %q", results[0].Interaction.Content)
	}
	// Second should be "sun is shining" (cosine~0.99)
	if results[1].Interaction.Content != "The sun is shining bright" {
		t.Errorf("expected 'The sun is shining bright', got %q", results[1].Interaction.Content)
	}
}

func TestInteractionStore_PersistAndReload(t *testing.T) {
	dir := t.TempDir()

	// Write data
	store := NewInteractionStore(dir)
	store.Open()
	store.Add(Interaction{
		ID:        "1",
		Role:      "user",
		Content:   "hello world",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, []float32{0.5, 0.5, 0.5})

	// Reload from disk
	store2 := NewInteractionStore(dir)
	if err := store2.Open(); err != nil {
		t.Fatal(err)
	}

	if store2.Count() != 1 {
		t.Fatalf("expected 1 after reload, got %d", store2.Count())
	}

	results := store2.Search([]float32{0.5, 0.5, 0.5}, 1, 0.0)
	if len(results) != 1 || results[0].Interaction.Content != "hello world" {
		t.Fatal("reload failed to preserve data")
	}
}

func TestInteractionStore_EmptySearch(t *testing.T) {
	store := NewInteractionStore(t.TempDir())
	store.Open()

	results := store.Search([]float32{1, 0}, 5, 0.0)
	if len(results) != 0 {
		t.Fatalf("expected 0 results from empty store, got %d", len(results))
	}
}

func TestInteractionStore_MinSimilarityFilter(t *testing.T) {
	store := NewInteractionStore(t.TempDir())
	store.Open()

	store.Add(Interaction{ID: "1", Content: "x", Timestamp: time.Now()},
		[]float32{1, 0, 0})
	store.Add(Interaction{ID: "2", Content: "y", Timestamp: time.Now()},
		[]float32{0, 1, 0})

	// Query orthogonal to "y" — similarity ~0
	results := store.Search([]float32{1, 0, 0}, 10, 0.9)
	if len(results) != 1 {
		t.Fatalf("expected 1 result above 0.9 threshold, got %d", len(results))
	}
	if results[0].Interaction.ID != "1" {
		t.Errorf("expected ID '1', got %q", results[0].Interaction.ID)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		a, b []float32
		want float64
	}{
		{[]float32{1, 0}, []float32{1, 0}, 1.0},
		{[]float32{1, 0}, []float32{0, 1}, 0.0},
		{[]float32{1, 0}, []float32{-1, 0}, -1.0},
		{[]float32{}, []float32{}, 0.0},
		{[]float32{0, 0}, []float32{1, 0}, 0.0},
	}

	for _, tt := range tests {
		got := cosineSimilarity(tt.a, tt.b)
		if math.Abs(got-tt.want) > 1e-6 {
			t.Errorf("cosine(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestEncodeDecodeEmbeddings(t *testing.T) {
	embeddings := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}

	data := encodeEmbeddings(embeddings)
	decoded := decodeEmbeddings(data, 3)

	if len(decoded) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(decoded))
	}
	for i := range embeddings {
		for j := range embeddings[i] {
			if decoded[i][j] != embeddings[i][j] {
				t.Errorf("mismatch at [%d][%d]: got %f, want %f", i, j, decoded[i][j], embeddings[i][j])
			}
		}
	}
}

func TestInteractionStore_InconsistentFiles(t *testing.T) {
	dir := t.TempDir()

	// Write 2 messages but only 1 embedding
	store := NewInteractionStore(dir)
	store.Open()
	store.Add(Interaction{ID: "1", Content: "a", Timestamp: time.Now()}, []float32{1, 0})
	store.Add(Interaction{ID: "2", Content: "b", Timestamp: time.Now()}, []float32{0, 1})

	// Corrupt: truncate embeddings.bin to hold only 1 vector (8 bytes for dim=2)
	embPath := dir + "/embeddings.bin"
	data, _ := os.ReadFile(embPath)
	os.WriteFile(embPath, data[:8], 0o600) // only 1 vector

	// Reload — should recover gracefully
	store2 := NewInteractionStore(dir)
	if err := store2.Open(); err != nil {
		t.Fatal(err)
	}
	if store2.Count() != 1 {
		t.Fatalf("expected 1 after recovery, got %d", store2.Count())
	}
}
