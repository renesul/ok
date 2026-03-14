package providers

import (
	"testing"

	"ok/internal/auth"
	"ok/internal/config"
)

func TestCreateProviderReturnsCodexCliProviderForCodexCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-codex"
	cfg.ProviderList = []config.ProviderConfig{
		{Name: "codex-cli", Workspace: "/tmp/workspace"},
	}
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-codex",
			Model:     "codex-cli/codex-model",
			Provider:  "codex-cli",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexCliProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeCliProviderForClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-claude-cli"
	cfg.ProviderList = []config.ProviderConfig{
		{Name: "claude-cli", Workspace: "/tmp/workspace"},
	}
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-claude-cli",
			Model:     "claude-cli/claude-sonnet",
			Provider:  "claude-cli",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeProviderForAnthropicOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "anthropic" {
			t.Fatalf("provider = %q, want anthropic", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "anthropic-token",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-claude-oauth"
	cfg.ProviderList = []config.ProviderConfig{
		{Name: "anthropic", AuthMethod: "oauth"},
	}
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-claude-oauth",
			Model:     "anthropic/claude-sonnet-4.6",
			Provider:  "anthropic",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeProvider", provider)
	}
}

func TestCreateProviderReturnsCodexProviderForOpenAIOAuth(t *testing.T) {
	// TODO: This test requires openai protocol to support auth_method: "oauth"
	// which is not yet implemented in the new factory_provider.go
	t.Skip("OpenAI OAuth via model_list not yet implemented")
}
