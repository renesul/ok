// Package webui provides an embedded web UI for managing OK configuration.
// It runs as an in-process HTTP server alongside the gateway, serving the
// config editor, auth flows, and log viewer on its own port.
package webui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"ok/internal/config"
	"ok/internal/logger"
)

//go:embed static/*
var staticFiles embed.FS

// reloadFunc is called when the web UI triggers a gateway reload.
var reloadFunc func()

// SetReloadFunc sets the callback that triggers a gateway reload.
func SetReloadFunc(fn func()) {
	reloadFunc = fn
}

// Start launches the web UI HTTP server on its own port.
// It is non-blocking — the server runs in a goroutine.
func Start(cfg config.WebUIConfig, configPath string) {
	mux := http.NewServeMux()

	registerConfigAPI(mux, configPath)
	registerAuthAPI(mux, configPath)
	registerGatewayAPI(mux, configPath)
	registerMCPAPI(mux)
	registerSkillsAPI(mux, configPath)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.ErrorC("webui", fmt.Sprintf("Failed to create static filesystem: %v", err))
		return
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.InfoCF("webui", "Web UI starting", map[string]any{
		"addr": addr,
	})
	fmt.Printf("✓ Web UI available at http://%s:%d/\n", cfg.Host, cfg.Port)
	if cfg.Host == "0.0.0.0" {
		if ip := GetLocalIP(); ip != "" {
			fmt.Printf("  Network: http://%s:%d/\n", ip, cfg.Port)
		}
	}

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.ErrorC("webui", fmt.Sprintf("Web UI server failed: %v", err))
		}
	}()
}
