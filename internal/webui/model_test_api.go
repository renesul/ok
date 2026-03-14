package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ok/internal/auth"
	"ok/internal/config"
	"ok/internal/logger"
	"ok/providers"
)

func registerModelTestAPI(mux *http.ServeMux, configPath string) {
	mux.HandleFunc("POST /api/models/test", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ModelIndex int `json:"model_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  fmt.Sprintf("failed to load config: %v", err),
			})
			return
		}

		if req.ModelIndex < 0 || req.ModelIndex >= len(cfg.ModelList) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  fmt.Sprintf("model_index %d out of range (have %d models)", req.ModelIndex, len(cfg.ModelList)),
			})
			return
		}

		modelCfg := cfg.ModelList[req.ModelIndex]
		modelName := modelCfg.Model

		provCfg, err := cfg.ResolveModelProvider(&modelCfg)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"model":  modelName,
				"error":  fmt.Sprintf("failed to resolve provider: %v", err),
			})
			return
		}

		// Embedding models use /v1/embeddings, not /v1/chat/completions
		if modelCfg.ModelName == "embedding" {
			testEmbeddingModel(w, r, &modelCfg, provCfg, modelName)
			return
		}

		// Transcription models use /v1/audio/transcriptions, not /v1/chat/completions
		if modelCfg.ModelName == "transcription" {
			testTranscriptionModel(w, r, &modelCfg, provCfg, modelName)
			return
		}

		logger.InfoCF("model-test", "Testing model", map[string]any{
			"model": modelName,
			"index": req.ModelIndex,
		})
		provider, modelID, err := providers.CreateProviderFromModelAndProvider(&modelCfg, provCfg)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"model":  modelName,
				"error":  fmt.Sprintf("failed to create provider: %v", err),
			})
			return
		}

		if sp, ok := provider.(providers.StatefulProvider); ok {
			defer sp.Close()
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		messages := []providers.Message{
			{Role: "user", Content: "Hi"},
		}

		resp, err := provider.Chat(ctx, messages, nil, modelID, nil)
		if err != nil {
			logger.ErrorC("model-test", fmt.Sprintf("%s: chat request failed: %v", modelName, err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"model":  modelName,
				"error":  fmt.Sprintf("chat request failed: %v", err),
			})
			return
		}

		logger.InfoCF("model-test", "Chat test passed", map[string]any{
			"model": modelName,
		})
		response := resp.Content
		if len(response) > 100 {
			response = response[:100] + "..."
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"model":    modelName,
			"response": response,
		})
	})
}

// testEmbeddingModel tests an embedding model by calling /v1/embeddings directly.
func testEmbeddingModel(w http.ResponseWriter, r *http.Request, modelCfg *config.ModelConfig, provCfg *config.ProviderConfig, modelName string) {
	apiBase := provCfg.APIBase
	if apiBase == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  "api_base is required for embedding model test",
		})
		return
	}

	// Extract the actual model ID (strip protocol prefix like "openai/")
	modelID := modelCfg.Model
	if parts := strings.SplitN(modelID, "/", 2); len(parts) == 2 {
		modelID = parts[1]
	}

	// Resolve API key: explicit > auth store
	apiKey := provCfg.APIKey
	if apiKey == "" {
		apiKey = resolveAPIKeyForTest(apiBase)
	}

	reqBody, _ := json.Marshal(map[string]any{
		"input":           "test",
		"model":           modelID,
		"encoding_format": "float",
	})

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(apiBase, "/v1")+"/v1/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("failed to create request: %v", err),
		})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorC("model-test", fmt.Sprintf("%s: embedding request failed: %v", modelName, err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("embedding request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorC("model-test", fmt.Sprintf("%s: embedding API error %d: %s", modelName, resp.StatusCode, string(body)))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("embedding API error %d: %s", resp.StatusCode, string(body)),
		})
		return
	}

	// Parse response to verify we got actual embeddings
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("failed to decode response: %v", err),
		})
		return
	}

	dim := 0
	if len(result.Data) > 0 {
		dim = len(result.Data[0].Embedding)
	}

	logger.InfoCF("model-test", "Embedding test passed", map[string]any{
		"model":      modelName,
		"dimensions": dim,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"model":    modelName,
		"response": fmt.Sprintf("embedding OK — %d dimensions", dim),
	})
}

// testTranscriptionModel tests a transcription model by verifying API connectivity
// via the /v1/models endpoint (since we can't easily generate test audio).
func testTranscriptionModel(w http.ResponseWriter, r *http.Request, modelCfg *config.ModelConfig, provCfg *config.ProviderConfig, modelName string) {
	apiBase := provCfg.APIBase
	if apiBase == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  "api_base is required for transcription model test",
		})
		return
	}

	// Extract the actual model ID (strip protocol prefix like "groq/")
	modelID := modelCfg.Model
	if parts := strings.SplitN(modelID, "/", 2); len(parts) == 2 {
		modelID = parts[1]
	}

	// Resolve API key: explicit > auth store
	apiKey := provCfg.APIKey
	if apiKey == "" {
		apiKey = resolveAPIKeyForTest(apiBase)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Call /v1/models/{modelID} to verify the API key and model exist
	url := strings.TrimSuffix(apiBase, "/v1") + "/v1/models/" + modelID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("failed to create request: %v", err),
		})
		return
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorC("model-test", fmt.Sprintf("%s: transcription API request failed: %v", modelName, err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("transcription API request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorC("model-test", fmt.Sprintf("%s: transcription API error %d: %s", modelName, resp.StatusCode, string(body)))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"model":  modelName,
			"error":  fmt.Sprintf("transcription API error %d: %s", resp.StatusCode, string(body)),
		})
		return
	}

	logger.InfoCF("model-test", "Transcription test passed", map[string]any{
		"model": modelName,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"model":    modelName,
		"response": "transcription model OK",
	})
}

// resolveAPIKeyForTest resolves API key from the auth store based on the base URL.
func resolveAPIKeyForTest(baseURL string) string {
	providerMap := map[string][]string{
		"openai":    {"api.openai.com"},
		"anthropic": {"api.anthropic.com"},
		"groq":      {"api.groq.com"},
		"deepseek":  {"api.deepseek.com"},
		"mistral":   {"api.mistral.ai"},
		"xai":       {"api.x.ai"},
	}
	for provider, hosts := range providerMap {
		for _, host := range hosts {
			if strings.Contains(baseURL, host) {
				cred, err := auth.GetCredential(provider)
				if err == nil && cred != nil && cred.AccessToken != "" {
					return cred.AccessToken
				}
			}
		}
	}
	return ""
}
