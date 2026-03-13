package appinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"ok/internal/config"
)

const Logo = "✓"

var (
	Version   = "dev"
	GitCommit string
	BuildTime string
	GoVersion string
)

// GetOKHome returns the ok home directory.
// Priority: $OK_HOME > ~/.ok
func GetOKHome() string {
	if home := os.Getenv("OK_HOME"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".ok")
	}
	return filepath.Join(home, ".ok")
}

func GetConfigPath() string {
	if configPath := os.Getenv("OK_CONFIG"); configPath != "" {
		return configPath
	}
	return filepath.Join(GetOKHome(), "config.json")
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := Version
	if GitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", GitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (string, string) {
	build := BuildTime
	goVer := GoVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return build, goVer
}
