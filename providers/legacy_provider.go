// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package providers

import (
	"fmt"

	"ok/internal/config"
)

// CreateProvider creates a provider based on the configuration.
// It uses the model_list configuration to create providers.
// Returns the provider, the model ID to use, and any error.
func CreateProvider(cfg *config.Config) (LLMProvider, string, error) {
	if len(cfg.ModelList) == 0 {
		return nil, "", fmt.Errorf("no providers configured. Please add entries to model_list in your config")
	}

	model := cfg.Agents.Defaults.GetModelName()

	// When no model is explicitly configured, pick the first non-embedding
	// entry from model_list so the assistant starts with a usable chat model.
	if model == "" {
		for _, m := range cfg.ModelList {
			if m.ModelName != "embedding" {
				model = m.ModelName
				break
			}
		}
		if model == "" {
			return nil, "", fmt.Errorf("no chat model found in model_list (only embedding models configured)")
		}
	}

	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, "", fmt.Errorf("model %q not found in model_list: %w", model, err)
	}

	if modelCfg.Workspace == "" {
		modelCfg.Workspace = cfg.WorkspacePath()
	}

	provider, modelID, err := CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create provider for model %q: %w", model, err)
	}

	return provider, modelID, nil
}
