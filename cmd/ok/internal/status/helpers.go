package status

import (
	"fmt"
	"os"

	"ok/cmd/ok/internal"
	"ok/internal/auth"
)

func statusCmd() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := internal.GetConfigPath()

	fmt.Printf("%s ok Status\n", internal.Logo)
	fmt.Printf("Version: %s\n", internal.FormatVersion())
	build, _ := internal.FormatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.GetModelName())

		// Check configured providers from model_list
		if len(cfg.ModelList) > 0 {
			fmt.Println("\nConfigured Models:")
			for _, m := range cfg.ModelList {
				if m.APIKey != "" || m.AuthMethod != "" {
					fmt.Printf("  %s: ✓\n", m.ModelName)
				} else {
					fmt.Printf("  %s: not set\n", m.ModelName)
				}
			}
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}
