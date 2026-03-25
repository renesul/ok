package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestEmbedding_Enabled(t *testing.T) {
	c := NewClient(ClientConfig{BaseURL: "http://x", Model: "m"}, zap.NewNop())
	if !c.Enabled() {
		t.Error("expected enabled with BaseURL+Model")
	}

	c2 := NewClient(ClientConfig{}, zap.NewNop())
	if c2.Enabled() {
		t.Error("expected disabled with empty config")
	}
}

func TestEmbedding_OpenAI_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openAIResponse{
			Data: []openAIEmbedding{
				{Embedding: []float32{0.1, 0.2, 0.3}},
				{Embedding: []float32{0.4, 0.5, 0.6}},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{Provider: "openai", BaseURL: srv.URL, Model: "test"}, zap.NewNop())
	result, err := c.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(result))
	}
	if len(result[0]) != 3 {
		t.Errorf("expected 3 dimensions, got %d", len(result[0]))
	}
}

func TestEmbedding_OpenAI_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, "rate limited")
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{Provider: "openai", BaseURL: srv.URL, Model: "test"}, zap.NewNop())
	_, err := c.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for 429")
	}
}

func TestEmbedding_OpenAI_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openAIResponse{Data: []openAIEmbedding{}})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{Provider: "openai", BaseURL: srv.URL, Model: "test"}, zap.NewNop())
	result, err := c.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 embeddings, got %d", len(result))
	}
}

func TestEmbedding_Disabled(t *testing.T) {
	c := NewClient(ClientConfig{}, zap.NewNop())
	_, err := c.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for disabled client")
	}
}
