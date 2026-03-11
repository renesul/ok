package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/renesul/ok/pkg/config"
	"github.com/renesul/ok/pkg/mcp"
)

func registerMCPAPI(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/mcp/test", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var cfg config.MCPServerConfig
		if err := json.Unmarshal(body, &cfg); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if cfg.Name == "" {
			http.Error(w, "Server name is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		tools, err := mcp.TestConnection(ctx, cfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Connection test failed: %v", err), http.StatusBadGateway)
			return
		}

		type toolInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		toolList := make([]toolInfo, 0, len(tools))
		for _, t := range tools {
			toolList = append(toolList, toolInfo{
				Name:        t.Name,
				Description: t.Description,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"tools":  toolList,
		})
	})
}
