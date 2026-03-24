package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type ClientConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

func (c ClientConfig) Enabled() bool {
	return c.BaseURL != "" && c.Model != ""
}

type Client struct {
	config     ClientConfig
	httpClient *http.Client
	log        *zap.Logger
}

func NewClient(config ClientConfig, log *zap.Logger) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{},
		log:        log.Named("embedding.client"),
	}
}

func (c *Client) Enabled() bool {
	return c.config.Enabled()
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("not configured")
	}
	_, err := c.Embed(ctx, []string{"ping"})
	return err
}

func (c *Client) Model() string {
	return c.config.Model
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("embedding not configured")
	}

	if c.config.Provider == "ollama" {
		return c.embedOllama(ctx, texts)
	}
	return c.embedOpenAI(ctx, texts)
}

func (c *Client) embedOpenAI(ctx context.Context, texts []string) ([][]float32, error) {
	url := strings.TrimRight(c.config.BaseURL, "/") + "/embeddings"

	body, _ := json.Marshal(map[string]interface{}{
		"model": c.config.Model,
		"input": texts,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	c.log.Debug("openai embedding request", zap.String("model", c.config.Model), zap.Int("texts", len(texts)))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding error %d: %s", resp.StatusCode, string(respBody))
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}

func (c *Client) embedOllama(ctx context.Context, texts []string) ([][]float32, error) {
	url := strings.TrimRight(c.config.BaseURL, "/") + "/api/embed"

	body, _ := json.Marshal(map[string]interface{}{
		"model": c.config.Model,
		"input": texts,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	c.log.Debug("ollama embedding request", zap.String("model", c.config.Model), zap.Int("texts", len(texts)))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding error %d: %s", resp.StatusCode, string(respBody))
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Embeddings, nil
}

type openAIResponse struct {
	Data []openAIEmbedding `json:"data"`
}

type openAIEmbedding struct {
	Embedding []float32 `json:"embedding"`
}

type ollamaResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}
