package webui

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func registerGatewayAPI(mux *http.ServeMux, absPath string) {
	logDir := logDirFromConfig(absPath)

	// GET /api/gateway/status — always running (web UI is in-process)
	mux.HandleFunc("GET /api/gateway/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "running",
		})
	})

	// POST /api/gateway/reload — trigger gateway restart with new config
	mux.HandleFunc("POST /api/gateway/reload", func(w http.ResponseWriter, r *http.Request) {
		if reloadFunc != nil {
			reloadFunc()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// DELETE /api/logs — delete all log files
	mux.HandleFunc("DELETE /api/logs", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(logDir)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			os.Truncate(filepath.Join(logDir, e.Name()), 0)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// GET /api/logs/components — list available log components
	mux.HandleFunc("GET /api/logs/components", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(logDir)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"components": []string{}})
			return
		}

		var components []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".log")
			if name == "all" {
				continue
			}
			components = append(components, name)
		}
		sort.Strings(components)
		result := append([]string{"all"}, components...)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"components": result})
	})

	// GET /api/logs/tail — last N lines from a component log file
	mux.HandleFunc("GET /api/logs/tail", func(w http.ResponseWriter, r *http.Request) {
		component := r.URL.Query().Get("component")
		if component == "" {
			component = "all"
		}

		for _, c := range component {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				http.Error(w, "invalid component name", http.StatusBadRequest)
				return
			}
		}

		maxLines := 500
		if v := r.URL.Query().Get("lines"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
				maxLines = n
			}
		}

		logPath := filepath.Join(logDir, component+".log")
		lines, err := tailFile(logPath, maxLines)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"logs": []any{}, "log_total": 0})
			return
		}

		jsonLogs := make([]json.RawMessage, 0, len(lines))
		for _, line := range lines {
			if json.Valid([]byte(line)) {
				jsonLogs = append(jsonLogs, json.RawMessage(line))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"logs":      jsonLogs,
			"log_total": len(jsonLogs),
		})
	})
}

func logDirFromConfig(absPath string) string {
	if home := os.Getenv("OK_HOME"); home != "" {
		return filepath.Join(home, "logs")
	}
	return filepath.Join(filepath.Dir(absPath), "logs")
}

func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, scanner.Err()
}

