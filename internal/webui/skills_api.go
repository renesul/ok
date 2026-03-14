package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"ok/internal/config"
	"ok/internal/logger"
	"ok/internal/skills"
)

func registerSkillsAPI(mux *http.ServeMux, configPath string) {
	// GET /api/skills — list installed skills
	mux.HandleFunc("GET /api/skills", func(w http.ResponseWriter, r *http.Request) {
		loader, _, _, err := buildSkillsDeps(configPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loader.ListSkills())
	})

	// GET /api/skills/show?name=X — show skill content
	mux.HandleFunc("GET /api/skills/show", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
			return
		}

		loader, _, _, err := buildSkillsDeps(configPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}

		content, ok := loader.LoadSkill(name)
		if !ok {
			http.Error(w, fmt.Sprintf("Skill '%s' not found", name), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"name":    name,
			"content": content,
		})
	})

	// POST /api/skills/remove — remove a skill
	mux.HandleFunc("POST /api/skills/remove", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Name == "" {
			http.Error(w, "Missing 'name' in request body", http.StatusBadRequest)
			return
		}

		_, installer, _, err := buildSkillsDeps(configPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}

		if err := installer.Uninstall(req.Name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove skill: %v", err), http.StatusInternalServerError)
			return
		}

		logger.InfoCF("webui.skills", "Skill removed", map[string]any{"name": req.Name})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// GET /api/skills/search?q=X&limit=N — search registry
	mux.HandleFunc("GET /api/skills/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing 'q' parameter", http.StatusBadRequest)
			return
		}

		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
			if limit <= 0 || limit > 50 {
				limit = 10
			}
		}

		_, _, regMgr, err := buildSkillsDeps(configPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		results, err := regMgr.SearchAll(ctx, query, limit)
		if err != nil {
			logger.WarnCF("webui.skills", "Skill search failed", map[string]any{"query": query, "error": err.Error()})
			http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	// POST /api/skills/install — install from registry
	mux.HandleFunc("POST /api/skills/install", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req struct {
			Slug     string `json:"slug"`
			Registry string `json:"registry"`
			Version  string `json:"version"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Slug == "" {
			http.Error(w, "Missing 'slug' in request body", http.StatusBadRequest)
			return
		}

		_, _, regMgr, err := buildSkillsDeps(configPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
			return
		}

		regName := req.Registry
		if regName == "" {
			regName = "clawhub"
		}
		reg := regMgr.GetRegistry(regName)
		if reg == nil {
			http.Error(w, fmt.Sprintf("Registry '%s' not found or not enabled", regName), http.StatusBadRequest)
			return
		}

		cfg2, _ := config.LoadConfig(configPath)
		targetDir := filepath.Join(cfg2.WorkspacePath(), "skills", req.Slug)

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		result, err := reg.DownloadAndInstall(ctx, req.Slug, req.Version, targetDir)
		if err != nil {
			logger.ErrorCF("webui.skills", "Skill install failed", map[string]any{"slug": req.Slug, "error": err.Error()})
			http.Error(w, fmt.Sprintf("Install failed: %v", err), http.StatusInternalServerError)
			return
		}

		logger.InfoCF("webui.skills", "Skill installed", map[string]any{
			"slug":    req.Slug,
			"version": result.Version,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":             "ok",
			"version":            result.Version,
			"is_malware_blocked": result.IsMalwareBlocked,
			"is_suspicious":      result.IsSuspicious,
			"summary":            result.Summary,
		})
	})
}

// buildSkillsDeps creates the loader, installer and registry manager from config.
func buildSkillsDeps(configPath string) (*skills.SkillsLoader, *skills.SkillInstaller, *skills.RegistryManager, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, nil, nil, err
	}

	workspace := cfg.WorkspacePath()
	globalDir := filepath.Dir(configPath)
	globalSkillsDir := filepath.Join(globalDir, "skills")
	builtinSkillsDir := filepath.Join(globalDir, "ok", "skills")

	loader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)
	installer := skills.NewSkillInstaller(workspace)

	regCfg := skills.RegistryConfig{
		ClawHub: skills.ClawHubConfig{
			Enabled:      cfg.Tools.Skills.Registries.ClawHub.Enabled,
			BaseURL:      cfg.Tools.Skills.Registries.ClawHub.BaseURL,
			AuthToken:    cfg.Tools.Skills.Registries.ClawHub.AuthToken,
			SearchPath:   cfg.Tools.Skills.Registries.ClawHub.SearchPath,
			SkillsPath:   cfg.Tools.Skills.Registries.ClawHub.SkillsPath,
			DownloadPath: cfg.Tools.Skills.Registries.ClawHub.DownloadPath,
			Timeout:      cfg.Tools.Skills.Registries.ClawHub.Timeout,
		},
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
	}
	regMgr := skills.NewRegistryManagerFromConfig(regCfg)

	return loader, installer, regMgr, nil
}
