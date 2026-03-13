package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ok/internal/config"
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

		provider, modelID, err := providers.CreateProviderFromConfig(&modelCfg)
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"model":  modelName,
				"error":  fmt.Sprintf("chat request failed: %v", err),
			})
			return
		}

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
