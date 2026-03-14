package webui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ok/internal/config"
	"ok/internal/logger"
)

func registerConfigAPI(mux *http.ServeMux, absPath string) {
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.LoadConfig(absPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"config": cfg,
			"path":   absPath,
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			logger.ErrorC("config-api", fmt.Sprintf("Failed to encode response: %v", err))
		}
	})

	mux.HandleFunc("PUT /api/config", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = config.WithConfigLock(func() error {
			var cfg config.Config
			if err := json.Unmarshal(body, &cfg); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return config.SaveConfig(absPath, &cfg)
		})
		if err != nil {
			logger.ErrorCF("config-api", "Config save failed", map[string]any{"error": err.Error()})
			http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
			return
		}

		logger.InfoC("config-api", "Config saved")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}
