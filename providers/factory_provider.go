// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package providers

import (
	"fmt"
	"strings"

	"ok/internal/auth"
	"ok/internal/config"
	"ok/internal/logger"
)

var getCredential = auth.GetCredential

// createClaudeAuthProvider creates a Claude provider using OAuth credentials from auth store.
func createClaudeAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("anthropic")
	if err != nil {
		logger.WarnCF("provider", "Claude credentials not found", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		logger.WarnC("provider", "No credentials for anthropic")
		return nil, fmt.Errorf("no credentials for anthropic. Run: ok auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

// createCodexAuthProvider creates a Codex provider using OAuth credentials from auth store.
func createCodexAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("openai")
	if err != nil {
		logger.WarnCF("provider", "Codex credentials not found", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		logger.WarnC("provider", "No credentials for openai")
		return nil, fmt.Errorf("no credentials for openai. Run: ok auth login --provider openai")
	}
	return NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource()), nil
}

// ExtractProtocol extracts the protocol prefix and model identifier from a model string.
// If no prefix is specified, it defaults to "openai".
// Examples:
//   - "openai/gpt-4o" -> ("openai", "gpt-4o")
//   - "anthropic/claude-sonnet-4.6" -> ("anthropic", "claude-sonnet-4.6")
//   - "gpt-4o" -> ("openai", "gpt-4o")  // default protocol
func ExtractProtocol(model string) (protocol, modelID string) {
	model = strings.TrimSpace(model)
	protocol, modelID, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, modelID
}

// CreateProviderFromModelAndProvider creates a provider based on the ModelConfig and ProviderConfig.
// Connectivity (api_base, api_key, auth_method, connect_mode, workspace) comes from the ProviderConfig.
// Model-specific settings (max_tokens_field, request_timeout) come from the ModelConfig.
// Returns the provider, the model ID (without protocol prefix), and any error.
func CreateProviderFromModelAndProvider(model *config.ModelConfig, prov *config.ProviderConfig) (LLMProvider, string, error) {
	if model == nil {
		return nil, "", fmt.Errorf("model config is nil")
	}
	if prov == nil {
		return nil, "", fmt.Errorf("provider config is nil")
	}
	if model.Model == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	protocol, modelID := ExtractProtocol(model.Model)
	logger.DebugCF("provider", "Creating provider from model+provider config", map[string]any{
		"protocol": protocol, "model_id": modelID, "provider": prov.Name,
	})

	apiBase := prov.APIBase
	apiKey := prov.APIKey
	authMethod := prov.AuthMethod
	connectMode := prov.ConnectMode
	workspace := prov.Workspace

	switch protocol {
	case "openai":
		if (authMethod == "oauth" || authMethod == "token") && apiKey == "" {
			provider, err := createCodexAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		if apiKey == "" && apiBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		if apiBase == "" {
			apiBase = GetDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			apiKey, apiBase, "", model.MaxTokensField, model.RequestTimeout,
		), modelID, nil

	case "anthropic":
		if (authMethod == "oauth" || authMethod == "token") && apiKey == "" {
			provider, err := createClaudeAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if apiKey == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic protocol (model: %s)", model.Model)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			apiKey, apiBase, "", model.MaxTokensField, model.RequestTimeout,
		), modelID, nil

	case "antigravity":
		return NewAntigravityProvider(), modelID, nil

	case "claude-cli", "claudecli":
		if workspace == "" {
			workspace = "."
		}
		return NewClaudeCliProvider(workspace), modelID, nil

	case "codex-cli", "codexcli":
		if workspace == "" {
			workspace = "."
		}
		return NewCodexCliProvider(workspace), modelID, nil

	case "github-copilot", "copilot":
		if apiBase == "" {
			apiBase = "localhost:4321"
		}
		if connectMode == "" {
			connectMode = "grpc"
		}
		provider, err := NewGitHubCopilotProvider(apiBase, connectMode, modelID)
		if err != nil {
			return nil, "", err
		}
		return provider, modelID, nil

	default:
		if authMethod == "token" && apiKey == "" {
			if apiBase == "" {
				apiBase = GetDefaultAPIBase(protocol)
			}
			if apiBase == "" {
				if cred, err := getCredential(protocol); err == nil && cred != nil && cred.APIBase != "" {
					apiBase = cred.APIBase
				}
			}
			if apiBase == "" {
				return nil, "", fmt.Errorf("no default API base for protocol %q; please set api_base in provider_list", protocol)
			}
			provider, err := createTokenAuthProvider(protocol, apiBase)
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		if apiKey == "" && apiBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		if apiBase == "" {
			apiBase = GetDefaultAPIBase(protocol)
		}
		if apiBase == "" {
			return nil, "", fmt.Errorf("no default API base for protocol %q; please set api_base in provider_list", protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			apiKey, apiBase, "", model.MaxTokensField, model.RequestTimeout,
		), modelID, nil
	}
}

// CreateProviderFromConfig creates a provider based on the ModelConfig.
// DEPRECATED: Use CreateProviderFromModelAndProvider instead.
// This is kept for backward compatibility and creates a synthetic ProviderConfig from the model.
func CreateProviderFromConfig(cfg *config.ModelConfig) (LLMProvider, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config is nil")
	}
	// Build a synthetic ProviderConfig from the model's protocol
	protocol, _ := ExtractProtocol(cfg.Model)
	syntheticProvider := &config.ProviderConfig{
		Name: protocol,
	}
	return CreateProviderFromModelAndProvider(cfg, syntheticProvider)
}

// GetDefaultAPIBase returns the default API base URL for a given protocol.
func GetDefaultAPIBase(protocol string) string {
	switch protocol {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "ollama":
		return "http://localhost:11434/v1"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "mistral":
		return "https://api.mistral.ai/v1"
	case "xai":
		return "https://api.x.ai/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai/"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "cerebras":
		return "https://api.cerebras.ai/v1"
	case "together":
		return "https://api.together.xyz/v1"
	case "qwen", "qwen-portal":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	default:
		return ""
	}
}

// createTokenAuthProvider creates an HTTP provider using token credentials from the auth store.
func createTokenAuthProvider(providerName, apiBase string) (LLMProvider, error) {
	cred, err := getCredential(providerName)
	if err != nil {
		logger.WarnCF("provider", "Token credentials not found", map[string]any{"provider": providerName, "error": err.Error()})
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		logger.WarnCF("provider", "No credentials found", map[string]any{"provider": providerName})
		return nil, fmt.Errorf("no credentials for %s. Run: ok auth login --provider %s", providerName, providerName)
	}
	return NewHTTPProvider(cred.AccessToken, apiBase, ""), nil
}
