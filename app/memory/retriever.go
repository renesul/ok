// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ok/internal/logger"
)

// Retriever provides semantic search over stored interactions.
type Retriever struct {
	store          *InteractionStore
	embedder       Embedder
	topK           int
	minScore       float64
	embeddingCache *EmbeddingCache
}

// NewRetriever creates a retriever with the given store and embedder.
func NewRetriever(store *InteractionStore, embedder Embedder, topK int, minScore float64) *Retriever {
	if topK <= 0 {
		topK = 5
	}
	if minScore <= 0 {
		minScore = 0.5
	}
	return &Retriever{
		store:          store,
		embedder:       embedder,
		topK:           topK,
		minScore:       minScore,
		embeddingCache: NewEmbeddingCache(100, 10*time.Minute),
	}
}

// Search finds the most relevant past interactions for a query.
func (r *Retriever) Search(ctx context.Context, query string) ([]ScoredInteraction, error) {
	if r.store.Count() == 0 {
		return nil, nil
	}

	// Check embedding cache first
	var queryEmb []float32
	var err error
	cacheHit := false

	if r.embeddingCache != nil {
		if cached, ok := r.embeddingCache.Get(query); ok {
			queryEmb = cached
			cacheHit = true
		}
	}

	if queryEmb == nil {
		queryEmb, err = r.embedder.Embed(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("embed query: %w", err)
		}
		if r.embeddingCache != nil {
			r.embeddingCache.Put(query, queryEmb)
		}
	}

	results := r.store.Search(queryEmb, r.topK, r.minScore)

	logger.DebugCF("rag", "Search completed", map[string]any{
		"query_len":       len(query),
		"results":         len(results),
		"store_count":     r.store.Count(),
		"embedding_cache": cacheHit,
	})

	return results, nil
}

// Index stores a new interaction with its embedding.
func (r *Retriever) Index(ctx context.Context, interaction Interaction) error {
	if strings.TrimSpace(interaction.Content) == "" {
		return nil
	}

	embedding, err := r.embedder.Embed(ctx, interaction.Content)
	if err != nil {
		return fmt.Errorf("embed interaction: %w", err)
	}

	if err := r.store.Add(interaction, embedding); err != nil {
		return fmt.Errorf("store interaction: %w", err)
	}

	logger.DebugCF("rag", "Indexed interaction", map[string]any{
		"id":   interaction.ID,
		"role": interaction.Role,
		"len":  len(interaction.Content),
	})

	return nil
}

// FormatContext formats retrieved interactions as context for the LLM prompt.
func FormatContext(results []ScoredInteraction) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Past Interactions\n\n")
	for i, r := range results {
		ts := r.Interaction.Timestamp.Format(time.DateOnly)
		fmt.Fprintf(&sb, "%d. [%s] %s: %s\n",
			i+1, ts, r.Interaction.Role,
			truncate(r.Interaction.Content, 500))
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
