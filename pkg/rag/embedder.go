// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder produces vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Model() string
}

// HTTPEmbedder calls an OpenAI-compatible /v1/embeddings endpoint.
// Works with OpenAI, Ollama, vLLM, LiteLLM, etc.
type HTTPEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewHTTPEmbedder creates an embedder that calls baseURL/v1/embeddings.
func NewHTTPEmbedder(baseURL, apiKey, model string) *HTTPEmbedder {
	return &HTTPEmbedder{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *HTTPEmbedder) Model() string { return e.model }

func (e *HTTPEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	batch, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, fmt.Errorf("embedding returned no results")
	}
	return batch[0], nil
}

// embeddingRequest is the OpenAI-compatible request body.
type embeddingRequest struct {
	Input          any    `json:"input"` // string or []string
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

// embeddingResponse is the OpenAI-compatible response body.
type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (e *HTTPEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	var input any
	if len(texts) == 1 {
		input = texts[0]
	} else {
		input = texts
	}

	reqBody := embeddingRequest{
		Input:          input,
		Model:          e.model,
		EncodingFormat: "float",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	url := e.baseURL + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}

	return embeddings, nil
}
