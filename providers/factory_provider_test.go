// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ok/internal/config"
)

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProtocol string
		wantModelID  string
	}{
		{
			name:         "openai with prefix",
			model:        "openai/gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "anthropic with prefix",
			model:        "anthropic/claude-sonnet-4.6",
			wantProtocol: "anthropic",
			wantModelID:  "claude-sonnet-4.6",
		},
		{
			name:         "no prefix - defaults to openai",
			model:        "gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "ollama with prefix",
			model:        "ollama/llama3",
			wantProtocol: "ollama",
			wantModelID:  "llama3",
		},
		{
			name:         "empty string",
			model:        "",
			wantProtocol: "openai",
			wantModelID:  "",
		},
		{
			name:         "with whitespace",
			model:        "  openai/gpt-4  ",
			wantProtocol: "openai",
			wantModelID:  "gpt-4",
		},
		{
			name:         "multiple slashes",
			model:        "nvidia/meta/llama-3.1-8b",
			wantProtocol: "nvidia",
			wantModelID:  "meta/llama-3.1-8b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, modelID := ExtractProtocol(tt.model)
			if protocol != tt.wantProtocol {
				t.Errorf("ExtractProtocol(%q) protocol = %q, want %q", tt.model, protocol, tt.wantProtocol)
			}
			if modelID != tt.wantModelID {
				t.Errorf("ExtractProtocol(%q) modelID = %q, want %q", tt.model, modelID, tt.wantModelID)
			}
		})
	}
}

func TestCreateProviderFromModelAndProvider_OpenAI(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-openai",
		Model:     "openai/gpt-4o",
		Provider:  "openai",
	}
	prov := &config.ProviderConfig{
		Name:   "openai",
		APIKey: "test-key",
		APIBase: "https://api.example.com/v1",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "gpt-4o" {
		t.Errorf("modelID = %q, want %q", modelID, "gpt-4o")
	}
}

func TestCreateProviderFromModelAndProvider_DefaultAPIBase(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"openai", "openai"},
		{"ollama", "ollama"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/test-model",
			}
			prov := &config.ProviderConfig{
				Name:   tt.protocol,
				APIKey: "test-key",
			}

			provider, _, err := CreateProviderFromModelAndProvider(model, prov)
			if err != nil {
				t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
			}

			// Verify we got an HTTPProvider for all these protocols
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestCreateProviderFromModelAndProvider_GenericHTTPProtocol(t *testing.T) {
	// Any unknown protocol with api_base should work as OpenAI-compatible
	model := &config.ModelConfig{
		ModelName: "test-custom",
		Model:     "deepseek/deepseek-chat",
	}
	prov := &config.ProviderConfig{
		Name:    "deepseek",
		APIKey:  "test-key",
		APIBase: "https://api.deepseek.com/v1",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "deepseek-chat" {
		t.Errorf("modelID = %q, want %q", modelID, "deepseek-chat")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestCreateProviderFromModelAndProvider_Anthropic(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-anthropic",
		Model:     "anthropic/claude-sonnet-4.6",
	}
	prov := &config.ProviderConfig{
		Name:   "anthropic",
		APIKey: "test-key",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
}

func TestCreateProviderFromModelAndProvider_Antigravity(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-antigravity",
		Model:     "antigravity/gemini-2.0-flash",
	}
	prov := &config.ProviderConfig{
		Name: "antigravity",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "gemini-2.0-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.0-flash")
	}
}

func TestCreateProviderFromModelAndProvider_ClaudeCLI(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-claude-cli",
		Model:     "claude-cli/claude-sonnet-4.6",
	}
	prov := &config.ProviderConfig{
		Name: "claude-cli",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
}

func TestCreateProviderFromModelAndProvider_CodexCLI(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-codex-cli",
		Model:     "codex-cli/codex",
	}
	prov := &config.ProviderConfig{
		Name: "codex-cli",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "codex" {
		t.Errorf("modelID = %q, want %q", modelID, "codex")
	}
}

func TestCreateProviderFromModelAndProvider_MissingAPIKey(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-no-key",
		Model:     "openai/gpt-4o",
	}
	prov := &config.ProviderConfig{
		Name: "openai",
	}

	_, _, err := CreateProviderFromModelAndProvider(model, prov)
	if err == nil {
		t.Fatal("CreateProviderFromModelAndProvider() expected error for missing API key")
	}
}

func TestCreateProviderFromModelAndProvider_UnknownProtocolNoAPIBase(t *testing.T) {
	model := &config.ModelConfig{
		ModelName: "test-unknown",
		Model:     "unknown-protocol/model",
	}
	prov := &config.ProviderConfig{
		Name:   "unknown-protocol",
		APIKey: "test-key",
	}

	_, _, err := CreateProviderFromModelAndProvider(model, prov)
	if err == nil {
		t.Fatal("CreateProviderFromModelAndProvider() expected error for unknown protocol without api_base")
	}
}

func TestCreateProviderFromModelAndProvider_UnknownProtocolWithAPIBase(t *testing.T) {
	// Unknown protocols should work if api_base is provided
	model := &config.ModelConfig{
		ModelName: "test-custom-provider",
		Model:     "custom-provider/some-model",
	}
	prov := &config.ProviderConfig{
		Name:    "custom-provider",
		APIKey:  "test-key",
		APIBase: "https://api.custom.com/v1",
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromModelAndProvider() returned nil provider")
	}
	if modelID != "some-model" {
		t.Errorf("modelID = %q, want %q", modelID, "some-model")
	}
}

func TestCreateProviderFromConfig_NilConfig(t *testing.T) {
	_, _, err := CreateProviderFromConfig(nil)
	if err == nil {
		t.Fatal("CreateProviderFromConfig(nil) expected error")
	}
}

func TestCreateProviderFromConfig_EmptyModel(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-empty",
		Model:     "",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for empty model")
	}
}

func TestCreateProviderFromModelAndProvider_RequestTimeoutPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	model := &config.ModelConfig{
		ModelName:      "test-timeout",
		Model:          "openai/gpt-4o",
		RequestTimeout: 1,
	}
	prov := &config.ProviderConfig{
		Name:    "openai",
		APIBase: server.URL,
	}

	provider, modelID, err := CreateProviderFromModelAndProvider(model, prov)
	if err != nil {
		t.Fatalf("CreateProviderFromModelAndProvider() error = %v", err)
	}
	if modelID != "gpt-4o" {
		t.Fatalf("modelID = %q, want %q", modelID, "gpt-4o")
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err == nil {
		t.Fatal("Chat() expected timeout error, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "context deadline exceeded") && !strings.Contains(errMsg, "Client.Timeout exceeded") {
		t.Fatalf("Chat() error = %q, want timeout-related error", errMsg)
	}
}
