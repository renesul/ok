package tools

import (
	"encoding/json"
	"fmt"

	"github.com/renesul/ok/domain"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxHTTPResponseBody = 2000

type httpRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type HTTPTool struct {
	client *http.Client
}

func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *HTTPTool) Name() string        { return "http" }
func (t *HTTPTool) Description() string { return "faz requisicao HTTP (GET/POST/PUT/DELETE com headers e body)" }
func (t *HTTPTool) Safety() domain.ToolSafety          { return domain.ToolRestricted }

func (t *HTTPTool) Run(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("input vazio")
	}

	var req httpRequest

	// Se input nao e JSON, trata como URL (GET simples)
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		req = httpRequest{Method: "GET", URL: input}
	}

	if req.URL == "" {
		return "", fmt.Errorf("url vazia")
	}

	if req.Method == "" {
		req.Method = "GET"
	}

	method := strings.ToUpper(req.Method)

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(method, req.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("criar request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if req.Body != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", method, req.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ler body: %w", err)
	}

	result := string(body)
	if len(result) > maxHTTPResponseBody {
		result = result[:maxHTTPResponseBody] + "..."
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, result)
	}

	return result, nil
}
