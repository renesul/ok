package startup

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"ok/internal/appinfo"
	"ok/internal/config"
)

//go:generate cp -r ../../workspace .

//go:embed workspace
var embeddedFiles embed.FS

// EnsureOnboarded creates default config and workspace templates if they don't exist.
func EnsureOnboarded() {
	configPath := appinfo.GetConfigPath()

	// If config already exists, nothing to do
	if _, err := os.Stat(configPath); err == nil {
		return
	}

	fmt.Println("First run detected — creating default config and workspace...")

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Warning: could not save default config: %v\n", err)
		return
	}

	workspace := cfg.WorkspacePath()
	if err := copyEmbeddedWorkspace(workspace); err != nil {
		fmt.Printf("Warning: could not create workspace templates: %v\n", err)
	}

	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Workspace: %s\n", workspace)
}

func copyEmbeddedWorkspace(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	return fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		relPath, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		targetPath := filepath.Join(targetDir, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		return os.WriteFile(targetPath, data, 0o644)
	})
}
