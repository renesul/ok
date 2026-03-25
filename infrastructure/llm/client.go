package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/security"
	"go.uber.org/zap"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClientConfig struct {
	BaseURL          string
	APIKey           string
	Model            string
	Temperature      float64
	MaxTokens        int
	MaxContextTokens int // limite da janela de contexto do modelo (0 = sem limite)
}

type StreamCallback func(token string) error

type Client struct {
	httpClient *http.Client
	scrubber   *security.SecretScrubber
	log        *zap.Logger
}

func NewClient(log *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		log:        log.Named("llm.client"),
	}
}

func (c *Client) SetScrubber(s *security.SecretScrubber) {
	c.scrubber = s
}

func (c *Client) scrubText(text string) string {
	if c.scrubber == nil {
		return text
	}
	return c.scrubber.Scrub(text)
}

func (c *Client) scrubMessages(messages []Message) []Message {
	if c.scrubber == nil {
		return messages
	}
	cleaned := make([]Message, len(messages))
	for i, msg := range messages {
		cleaned[i] = Message{Role: msg.Role, Content: c.scrubber.Scrub(msg.Content)}
	}
	return cleaned
}

func (c *Client) ChatCompletionStream(ctx context.Context, config ClientConfig, messages []Message, onToken StreamCallback) (string, error) {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	body := map[string]interface{}{
		"model":       config.Model,
		"messages":    c.scrubMessages(messages),
		"stream":      true,
		"temperature": config.Temperature,
	}
	if config.MaxTokens > 0 {
		body["max_tokens"] = config.MaxTokens
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	c.log.Debug("llm request", zap.String("url", url), zap.String("model", config.Model))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm error %d: %s", resp.StatusCode, string(respBody))
	}

	return c.processStream(resp.Body, onToken)
}

// ChatCompletionSync executa uma completion nao-streaming e retorna o texto completo
func (c *Client) ChatCompletionSync(ctx context.Context, config ClientConfig, messages []Message) (string, error) {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	body := map[string]interface{}{
		"model":       config.Model,
		"messages":    c.scrubMessages(messages),
		"stream":      false,
		"temperature": config.Temperature,
	}
	if config.MaxTokens > 0 {
		body["max_tokens"] = config.MaxTokens
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm sync request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

func (c *Client) processStream(body io.Reader, onToken StreamCallback) (string, error) {
	scanner := bufio.NewScanner(body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.log.Debug("skip unparseable chunk", zap.Error(err))
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		token := chunk.Choices[0].Delta.Content
		if token == "" {
			continue
		}

		fullResponse.WriteString(token)
		if err := onToken(token); err != nil {
			return fullResponse.String(), fmt.Errorf("token callback: %w", err)
		}
	}

	return fullResponse.String(), scanner.Err()
}

func (c *Client) Ping(ctx context.Context, config ClientConfig) error {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	body := map[string]interface{}{
		"model":      config.Model,
		"messages":   []Message{{Role: "user", Content: "hi"}},
		"max_tokens": 1,
		"stream":     false,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) Decide(ctx context.Context, config ClientConfig, systemPrompt, userPrompt string) (domain.Decision, error) {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	messages := []Message{
		{Role: "system", Content: c.scrubText(systemPrompt)},
		{Role: "user", Content: c.scrubText(userPrompt)},
	}

	body := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"temperature": 0.2,
		"max_tokens":  500,
		"stream":      false,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.Decision{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	c.log.Debug("llm decide", zap.String("model", config.Model))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("decide request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return domain.Decision{}, fmt.Errorf("decide error %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.Decision{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return domain.Decision{Done: true}, nil
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)

	// Parse flexivel: input pode ser string ou objeto JSON
	var flexible struct {
		Tool  string          `json:"tool"`
		Input json.RawMessage `json:"input"`
		Done  bool            `json:"done"`
	}
	if err := json.Unmarshal([]byte(content), &flexible); err != nil {
		c.log.Debug("decision parse fallback", zap.String("content", content))
		return domain.Decision{Input: content, Done: true}, nil
	}

	var inputStr string
	if err := json.Unmarshal(flexible.Input, &inputStr); err != nil {
		// Input e objeto JSON — serializar como string
		inputStr = string(flexible.Input)
	}

	return domain.Decision{
		Tool:  flexible.Tool,
		Input: inputStr,
		Done:  flexible.Done,
	}, nil
}

func (c *Client) CreatePlanStreaming(ctx context.Context, config ClientConfig, systemPrompt, goal string, onToken StreamCallback) (domain.ExecutionPlan, error) {
	messages := []Message{
		{Role: "system", Content: c.scrubText(systemPrompt)},
		{Role: "user", Content: c.scrubText(goal)},
	}

	cfg := config
	cfg.Temperature = 0.2
	cfg.MaxTokens = 800

	fullText, err := c.ChatCompletionStream(ctx, cfg, messages, onToken)
	if err != nil {
		return domain.ExecutionPlan{}, fmt.Errorf("plan stream: %w", err)
	}

	content := strings.TrimSpace(fullText)
	var plan domain.ExecutionPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		c.log.Debug("plan parse failed", zap.String("content", content), zap.Error(err))
		return domain.ExecutionPlan{}, nil
	}
	return plan, nil
}

func (c *Client) CreatePlan(ctx context.Context, config ClientConfig, systemPrompt, goal string) (domain.ExecutionPlan, error) {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	messages := []Message{
		{Role: "system", Content: c.scrubText(systemPrompt)},
		{Role: "user", Content: c.scrubText(goal)},
	}

	body := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"temperature": 0.2,
		"max_tokens":  800,
		"stream":      false,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.ExecutionPlan{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	c.log.Debug("llm create plan", zap.String("model", config.Model))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.ExecutionPlan{}, fmt.Errorf("plan request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return domain.ExecutionPlan{}, fmt.Errorf("plan error %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.ExecutionPlan{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return domain.ExecutionPlan{}, nil
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)

	var plan domain.ExecutionPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		c.log.Debug("plan parse failed", zap.String("content", content), zap.Error(err))
		return domain.ExecutionPlan{}, fmt.Errorf("plan parse error: %w", err)
	}

	return plan, nil
}

func (c *Client) Reflect(ctx context.Context, config ClientConfig, systemPrompt, executionContext string) (domain.ReflectionResult, error) {
	url := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"

	messages := []Message{
		{Role: "system", Content: c.scrubText(systemPrompt)},
		{Role: "user", Content: c.scrubText(executionContext)},
	}

	body := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"temperature": 0.1,
		"max_tokens":  300,
		"stream":      false,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.ReflectionResult{Action: "continue"}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	c.log.Debug("llm reflect", zap.String("model", config.Model))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.ReflectionResult{}, fmt.Errorf("reflect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return domain.ReflectionResult{}, fmt.Errorf("reflect error %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.ReflectionResult{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return domain.ReflectionResult{}, fmt.Errorf("no choices in response")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)

	var reflection domain.ReflectionResult
	if err := json.Unmarshal([]byte(content), &reflection); err != nil {
		c.log.Debug("reflect parse fallback", zap.String("content", content), zap.Error(err))
		return domain.ReflectionResult{}, fmt.Errorf("reflect parse error: %w", err)
	}

	return reflection, nil
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatMessage struct {
	Content string `json:"content"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamDelta struct {
	Content string `json:"content"`
}
