package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort   string `mapstructure:"SERVER_PORT"`
	DatabasePath string `mapstructure:"DATABASE_PATH"`
	LogLevel     string `mapstructure:"LOG_LEVEL"`
	Debug        bool   `mapstructure:"DEBUG"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DATABASE_PATH", "./data/ok.db")
	viper.SetDefault("LOG_LEVEL", "error")
	viper.SetDefault("DEBUG", false)

	// .env is optional — env vars take precedence
	_ = viper.ReadInConfig()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.Debug {
		cfg.LogLevel = "debug"
	}

	return cfg, nil
}
