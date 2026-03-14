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
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: ok auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

// createCodexAuthProvider creates a Codex provider using OAuth credentials from auth store.
func createCodexAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
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

// CreateProviderFromConfig creates a provider based on the ModelConfig.
// It uses the protocol prefix in the Model field to determine which provider to create.
// Supported protocols: openai, litellm, anthropic, antigravity, claude-cli, codex-cli, github-copilot
// Returns the provider, the model ID (without protocol prefix), and any error.
func CreateProviderFromConfig(cfg *config.ModelConfig) (LLMProvider, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config is nil")
	}

	if cfg.Model == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	protocol, modelID := ExtractProtocol(cfg.Model)
	logger.DebugCF("provider", "Creating provider from model config", map[string]any{"protocol": protocol, "model_id": modelID})

	switch protocol {
	case "openai":
		// OpenAI with OAuth/token auth (Codex-style) — only if no explicit API key
		if (cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token") && cfg.APIKey == "" {
			provider, err := createCodexAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// OpenAI with API key
		if cfg.APIKey == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = GetDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			"",
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil

	case "anthropic":
		if (cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token") && cfg.APIKey == "" {
			// Use OAuth credentials from auth store (only if no explicit API key)
			provider, err := createClaudeAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// Use API key with HTTP API
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if cfg.APIKey == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic protocol (model: %s)", cfg.Model)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			"",
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil

	case "antigravity":
		return NewAntigravityProvider(), modelID, nil

	case "claude-cli", "claudecli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewClaudeCliProvider(workspace), modelID, nil

	case "codex-cli", "codexcli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewCodexCliProvider(workspace), modelID, nil

	case "github-copilot", "copilot":
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "localhost:4321"
		}
		connectMode := cfg.ConnectMode
		if connectMode == "" {
			connectMode = "grpc"
		}
		provider, err := NewGitHubCopilotProvider(apiBase, connectMode, modelID)
		if err != nil {
			return nil, "", err
		}
		return provider, modelID, nil

	default:
		// Any other protocol is treated as OpenAI-compatible HTTP provider.
		// This allows model_list entries with any vendor prefix to work.
		if cfg.AuthMethod == "token" && cfg.APIKey == "" {
			apiBase := cfg.APIBase
			if apiBase == "" {
				apiBase = GetDefaultAPIBase(protocol)
			}
			// For custom providers, try the auth store's APIBase
			if apiBase == "" {
				if cred, err := getCredential(protocol); err == nil && cred != nil && cred.APIBase != "" {
					apiBase = cred.APIBase
				}
			}
			if apiBase == "" {
				return nil, "", fmt.Errorf("no default API base for protocol %q; please set api_base in model_list", protocol)
			}
			provider, err := createTokenAuthProvider(protocol, apiBase)
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		if cfg.APIKey == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = GetDefaultAPIBase(protocol)
		}
		if apiBase == "" {
			return nil, "", fmt.Errorf("no default API base for protocol %q; please set api_base in model_list", protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			"",
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil
	}
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
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for %s. Run: ok auth login --provider %s", providerName, providerName)
	}
	return NewHTTPProvider(cred.AccessToken, apiBase, ""), nil
}
