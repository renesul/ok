package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort   string `mapstructure:"SERVER_PORT"`
	DatabasePath string `mapstructure:"DATABASE_PATH"`
	LogLevel     string `mapstructure:"LOG_LEVEL"`
	LogPath      string `mapstructure:"LOG_PATH"`
	Debug        bool   `mapstructure:"DEBUG"`
	AuthPassword string `mapstructure:"AUTH_PASSWORD"`
	LLMBaseURL     string `mapstructure:"LLM_BASE_URL"`
	LLMAPIKey      string `mapstructure:"LLM_API_KEY"`
	LLMModel       string `mapstructure:"LLM_MODEL"`
	LLMFastBaseURL string `mapstructure:"LLM_FAST_BASE_URL"`
	LLMFastAPIKey  string `mapstructure:"LLM_FAST_API_KEY"`
	LLMFastModel   string `mapstructure:"LLM_FAST_MODEL"`
	EmbedProvider string `mapstructure:"EMBED_PROVIDER"`
	EmbedBaseURL  string `mapstructure:"EMBED_BASE_URL"`
	EmbedAPIKey   string `mapstructure:"EMBED_API_KEY"`
	EmbedModel      string `mapstructure:"EMBED_MODEL"`
	AgentSandboxDir      string `mapstructure:"AGENT_SANDBOX_DIR"`
	WhatsAppOwnerNumber  string `mapstructure:"WHATSAPP_OWNER_NUMBER"`
	WhatsAppDBPath       string `mapstructure:"WHATSAPP_DB_PATH"`
	TelegramBotToken     string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	TelegramOwnerID      int64  `mapstructure:"TELEGRAM_OWNER_ID"`
	DiscordBotToken      string `mapstructure:"DISCORD_BOT_TOKEN"`
	DiscordOwnerID       string `mapstructure:"DISCORD_OWNER_ID"`
}

func Load() (*Config, error) {
	return LoadFrom("data/.env")
}

func LoadFrom(envFile string) (*Config, error) {
	viper.SetConfigFile(envFile)
	if strings.Contains(envFile, ".env") {
		viper.SetConfigType("env")
	}
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DATABASE_PATH", "./data/ok.db")
	viper.SetDefault("LOG_LEVEL", "error")
	viper.SetDefault("LOG_PATH", "data/ok.log")
	viper.SetDefault("DEBUG", false)
	viper.SetDefault("AUTH_PASSWORD", "admin")
	viper.SetDefault("LLM_BASE_URL", "")
	viper.SetDefault("LLM_API_KEY", "")
	viper.SetDefault("LLM_MODEL", "")
	viper.SetDefault("LLM_FAST_BASE_URL", "")
	viper.SetDefault("LLM_FAST_API_KEY", "")
	viper.SetDefault("LLM_FAST_MODEL", "")
	viper.SetDefault("EMBED_PROVIDER", "")
	viper.SetDefault("EMBED_BASE_URL", "")
	viper.SetDefault("EMBED_API_KEY", "")
	viper.SetDefault("EMBED_MODEL", "")
	viper.SetDefault("AGENT_SANDBOX_DIR", "data/sandbox")
	viper.SetDefault("WHATSAPP_OWNER_NUMBER", "")
	viper.SetDefault("WHATSAPP_DB_PATH", "data/whatsapp.db")
	viper.SetDefault("TELEGRAM_BOT_TOKEN", "")
	viper.SetDefault("TELEGRAM_OWNER_ID", 0)
	viper.SetDefault("DISCORD_BOT_TOKEN", "")
	viper.SetDefault("DISCORD_OWNER_ID", "")

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
