package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/caarlos0/env/v11"

	"ok/internal/utils"
	"ok/internal/logger"
)

// configMu protects read-modify-write cycles on config.json against race conditions.
var configMu sync.Mutex

// WithConfigLock serializes read-modify-write operations on the config file.
func WithConfigLock(fn func() error) error {
	configMu.Lock()
	defer configMu.Unlock()
	return fn()
}

// rrCounter is a global counter for round-robin load balancing across models.
var rrCounter atomic.Uint64

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

type Config struct {
	Debug        bool             `json:"debug" env:"OK_DEBUG"`
	Agents       AgentsConfig     `json:"agents"`
	Session      SessionConfig    `json:"session,omitempty"`
	Channels     ChannelsConfig   `json:"channels"`
	ProviderList []ProviderConfig `json:"provider_list"`
	ModelList    []ModelConfig    `json:"model_list"`
	Gateway      GatewayConfig    `json:"gateway"`
	Tools        ToolsConfig      `json:"tools"`
	Heartbeat    HeartbeatConfig  `json:"heartbeat"`
	Devices      DevicesConfig    `json:"devices"`
	RAG          RAGConfig        `json:"rag,omitempty"`
	MCPServers   []MCPServerConfig `json:"mcp_servers,omitempty"`
	WebUI        WebUIConfig      `json:"web_ui,omitempty"`
	Integrations IntegrationsConfig `json:"integrations,omitempty"`
	Proxy        string           `json:"proxy,omitempty" env:"OK_PROXY"`
}

// IntegrationsConfig holds configuration for external service integrations.
type IntegrationsConfig struct {
	Email         EmailConfig         `json:"email,omitempty"`
	Calendar      CalendarConfig      `json:"calendar,omitempty"`
	HomeAssistant HomeAssistantConfig `json:"home_assistant,omitempty"`
}

// EmailConfig configures IMAP/SMTP email integration.
type EmailConfig struct {
	Enabled  bool   `json:"enabled"            env:"OK_EMAIL_ENABLED"`
	IMAPHost string `json:"imap_host"          env:"OK_EMAIL_IMAP_HOST"`
	IMAPPort int    `json:"imap_port"          env:"OK_EMAIL_IMAP_PORT"`
	IMAPTLS  bool   `json:"imap_tls"           env:"OK_EMAIL_IMAP_TLS"`
	SMTPHost string `json:"smtp_host"          env:"OK_EMAIL_SMTP_HOST"`
	SMTPPort int    `json:"smtp_port"          env:"OK_EMAIL_SMTP_PORT"`
	Username string `json:"username"           env:"OK_EMAIL_USERNAME"`
	Password string `json:"password"           env:"OK_EMAIL_PASSWORD"`
	FromName string `json:"from_name,omitempty" env:"OK_EMAIL_FROM_NAME"`
	MaxFetch int    `json:"max_fetch,omitempty"` // max emails per read, default 10
}

// CalendarConfig configures Google Calendar and Microsoft Outlook integration.
type CalendarConfig struct {
	Enabled bool `json:"enabled" env:"OK_CALENDAR_ENABLED"`

	// Google Calendar
	GoogleEnabled  bool   `json:"google_enabled"   env:"OK_CALENDAR_GOOGLE_ENABLED"`
	GoogleAPIKey   string `json:"google_api_key"   env:"OK_CALENDAR_GOOGLE_API_KEY"`
	GoogleCalendarID string `json:"google_calendar_id" env:"OK_CALENDAR_GOOGLE_CALENDAR_ID"` // default: "primary"

	// Microsoft Outlook (Graph API)
	OutlookEnabled      bool   `json:"outlook_enabled"       env:"OK_CALENDAR_OUTLOOK_ENABLED"`
	OutlookAccessToken  string `json:"outlook_access_token"  env:"OK_CALENDAR_OUTLOOK_ACCESS_TOKEN"`
	OutlookCalendarID   string `json:"outlook_calendar_id"   env:"OK_CALENDAR_OUTLOOK_CALENDAR_ID"` // default: "primary" (me/calendar)
}

// HomeAssistantConfig configures Home Assistant REST API integration.
type HomeAssistantConfig struct {
	Enabled bool   `json:"enabled" env:"OK_HA_ENABLED"`
	URL     string `json:"url"     env:"OK_HA_URL"`   // e.g. http://homeassistant.local:8123
	Token   string `json:"token"   env:"OK_HA_TOKEN"` // Long-lived access token
}

// WebUIConfig configures the embedded web UI served alongside the gateway.
type WebUIConfig struct {
	Enabled bool   `json:"enabled" env:"OK_WEBUI_ENABLED"`
	Host    string `json:"host"    env:"OK_WEBUI_HOST"`
	Port    int    `json:"port"    env:"OK_WEBUI_PORT"`
}

// RAGConfig configures retrieval-augmented generation for long-term memory.
type RAGConfig struct {
	Enabled       bool    `json:"enabled"         env:"OK_RAG_ENABLED"`
	BaseURL       string  `json:"base_url"        env:"OK_RAG_BASE_URL"`
	APIKey        string  `json:"api_key"         env:"OK_RAG_API_KEY"`
	Model         string  `json:"model"           env:"OK_RAG_MODEL"`
	TopK          int     `json:"top_k"           env:"OK_RAG_TOP_K"`
	MinSimilarity float64 `json:"min_similarity"  env:"OK_RAG_MIN_SIMILARITY"`
}

// MCPServerConfig configures an MCP (Model Context Protocol) server connection.
type MCPServerConfig struct {
	Name       string            `json:"name"`
	Enabled    bool              `json:"enabled"`
	Transport  string            `json:"transport"`              // "stdio" | "http" | "sse"
	Command    string            `json:"command,omitempty"`      // stdio only
	Args       []string          `json:"args,omitempty"`         // stdio only
	Env        map[string]string `json:"env,omitempty"`          // stdio only
	URL        string            `json:"url,omitempty"`          // http/sse only
	Headers    map[string]string `json:"headers,omitempty"`      // http/sse only
	Timeout    int               `json:"timeout,omitempty"`      // seconds, default 30
	ToolPrefix string            `json:"tool_prefix,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Config
// to omit session section when empty
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		Session *SessionConfig `json:"session,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(&c),
	}

	// Only include session if not empty
	if c.Session.DMScope != "" || len(c.Session.IdentityLinks) > 0 {
		aux.Session = &c.Session
	}

	return json.Marshal(aux)
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

// AgentModelConfig supports both string and structured model config.
// String format: "gpt-4" (just primary, no fallbacks)
// Object format: {"primary": "gpt-4", "fallbacks": ["claude-haiku"]}
type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		m.Fallbacks = nil
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

func (m AgentModelConfig) MarshalJSON() ([]byte, error) {
	if len(m.Fallbacks) == 0 && m.Primary != "" {
		return json.Marshal(m.Primary)
	}
	type raw struct {
		Primary   string   `json:"primary,omitempty"`
		Fallbacks []string `json:"fallbacks,omitempty"`
	}
	return json.Marshal(raw{Primary: m.Primary, Fallbacks: m.Fallbacks})
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type SessionConfig struct {
	DMScope       string              `json:"dm_scope,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
}

// RoutingConfig controls the intelligent model routing feature.
// When enabled, each incoming message is scored against structural features
// (message length, code blocks, tool call history, conversation depth, attachments).
// Messages scoring below Threshold are sent to LightModel; all others use the
// agent's primary model. This reduces cost and latency for simple tasks without
// requiring any keyword matching — all scoring is language-agnostic.
type RoutingConfig struct {
	Enabled    bool    `json:"enabled"`
	LightModel string  `json:"light_model"` // model_name from model_list to use for simple tasks
	Threshold  float64 `json:"threshold"`   // complexity score in [0,1]; score >= threshold → primary model
}

type AgentDefaults struct {
	Workspace                 string         `json:"workspace"                       env:"OK_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace       bool           `json:"restrict_to_workspace"           env:"OK_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	AllowReadOutsideWorkspace bool           `json:"allow_read_outside_workspace"    env:"OK_AGENTS_DEFAULTS_ALLOW_READ_OUTSIDE_WORKSPACE"`
	Provider                  string         `json:"provider"                        env:"OK_AGENTS_DEFAULTS_PROVIDER"`
	ModelName                 string         `json:"model_name,omitempty"            env:"OK_AGENTS_DEFAULTS_MODEL_NAME"`
	Model                     string         `json:"model"                           env:"OK_AGENTS_DEFAULTS_MODEL"` // Deprecated: use model_name instead
	ModelFallbacks            []string       `json:"model_fallbacks,omitempty"`
	ImageModel                string         `json:"image_model,omitempty"           env:"OK_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks       []string       `json:"image_model_fallbacks,omitempty"`
	MaxTokens                 int            `json:"max_tokens"                      env:"OK_AGENTS_DEFAULTS_MAX_TOKENS"`
	Temperature               *float64       `json:"temperature,omitempty"           env:"OK_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations         int            `json:"max_tool_iterations"             env:"OK_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	SummarizeMessageThreshold int            `json:"summarize_message_threshold"     env:"OK_AGENTS_DEFAULTS_SUMMARIZE_MESSAGE_THRESHOLD"`
	SummarizeTokenPercent     int            `json:"summarize_token_percent"         env:"OK_AGENTS_DEFAULTS_SUMMARIZE_TOKEN_PERCENT"`
	MaxMediaSize              int            `json:"max_media_size,omitempty"        env:"OK_AGENTS_DEFAULTS_MAX_MEDIA_SIZE"`
	Routing                   *RoutingConfig `json:"routing,omitempty"`
}

const DefaultMaxMediaSize = 20 * 1024 * 1024 // 20 MB

func (d *AgentDefaults) GetMaxMediaSize() int {
	if d.MaxMediaSize > 0 {
		return d.MaxMediaSize
	}
	return DefaultMaxMediaSize
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *AgentDefaults) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type ChatConfig struct {
	Enabled bool `json:"enabled" env:"OK_CHANNELS_CHAT_ENABLED"`
}

type ChannelsConfig struct {
	Chat     ChatConfig     `json:"chat"`
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Slack    SlackConfig    `json:"slack"`
}

// GroupTriggerConfig controls when the bot responds in group chats.
type GroupTriggerConfig struct {
	MentionOnly bool     `json:"mention_only,omitempty"`
	Prefixes    []string `json:"prefixes,omitempty"`
}

// TypingConfig controls typing indicator behavior (Phase 10).
type TypingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

// PlaceholderConfig controls placeholder message behavior (Phase 10).
type PlaceholderConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Text    string `json:"text,omitempty"`
}

type WhatsAppConfig struct {
	Enabled            bool                `json:"enabled"              env:"OK_CHANNELS_WHATSAPP_ENABLED"`
	SessionStorePath   string              `json:"session_store_path"   env:"OK_CHANNELS_WHATSAPP_SESSION_STORE_PATH"`
	AllowSelf          bool                `json:"allow_self"           env:"OK_CHANNELS_WHATSAPP_ALLOW_SELF"`
	AllowDirect        bool                `json:"allow_direct"         env:"OK_CHANNELS_WHATSAPP_ALLOW_DIRECT"`
	AllowGroups        bool                `json:"allow_groups"         env:"OK_CHANNELS_WHATSAPP_ALLOW_GROUPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"OK_CHANNELS_WHATSAPP_ALLOW_FROM"`
	AllowedGroups      FlexibleStringSlice `json:"allowed_groups"       env:"OK_CHANNELS_WHATSAPP_ALLOWED_GROUPS"`
	AllowedContacts    FlexibleStringSlice `json:"allowed_contacts"     env:"OK_CHANNELS_WHATSAPP_ALLOWED_CONTACTS"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"OK_CHANNELS_WHATSAPP_REASONING_CHANNEL_ID"`
}

type TelegramConfig struct {
	Enabled            bool                `json:"enabled"                 env:"OK_CHANNELS_TELEGRAM_ENABLED"`
	Token              string              `json:"token"                   env:"OK_CHANNELS_TELEGRAM_TOKEN"`
	BaseURL            string              `json:"base_url"                env:"OK_CHANNELS_TELEGRAM_BASE_URL"`
	AllowDirect        bool                `json:"allow_direct"            env:"OK_CHANNELS_TELEGRAM_ALLOW_DIRECT"`
	AllowGroups        bool                `json:"allow_groups"            env:"OK_CHANNELS_TELEGRAM_ALLOW_GROUPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"OK_CHANNELS_TELEGRAM_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"OK_CHANNELS_TELEGRAM_REASONING_CHANNEL_ID"`
}

type DiscordConfig struct {
	Enabled            bool                `json:"enabled"                 env:"OK_CHANNELS_DISCORD_ENABLED"`
	Token              string              `json:"token"                   env:"OK_CHANNELS_DISCORD_TOKEN"`
	AllowDirect        bool                `json:"allow_direct"            env:"OK_CHANNELS_DISCORD_ALLOW_DIRECT"`
	AllowGroups        bool                `json:"allow_groups"            env:"OK_CHANNELS_DISCORD_ALLOW_GROUPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"OK_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly        bool                `json:"mention_only"            env:"OK_CHANNELS_DISCORD_MENTION_ONLY"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"OK_CHANNELS_DISCORD_REASONING_CHANNEL_ID"`
}

type SlackConfig struct {
	Enabled            bool                `json:"enabled"                 env:"OK_CHANNELS_SLACK_ENABLED"`
	BotToken           string              `json:"bot_token"               env:"OK_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken           string              `json:"app_token"               env:"OK_CHANNELS_SLACK_APP_TOKEN"`
	AllowDirect        bool                `json:"allow_direct"            env:"OK_CHANNELS_SLACK_ALLOW_DIRECT"`
	AllowGroups        bool                `json:"allow_groups"            env:"OK_CHANNELS_SLACK_ALLOW_GROUPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"OK_CHANNELS_SLACK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"OK_CHANNELS_SLACK_REASONING_CHANNEL_ID"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled"  env:"OK_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"OK_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type DevicesConfig struct {
	Enabled    bool `json:"enabled"     env:"OK_DEVICES_ENABLED"`
	MonitorUSB bool `json:"monitor_usb" env:"OK_DEVICES_MONITOR_USB"`
}

// ProviderConfig represents a connectivity/authentication configuration for an LLM provider.
// Models reference a provider by name (FK) to inherit connectivity settings.
type ProviderConfig struct {
	Name        string `json:"name"`                     // "openai", "anthropic", etc.
	APIBase     string `json:"api_base,omitempty"`       // URL base
	APIKey      string `json:"api_key,omitempty"`        // Explicit API key (alternative to auth store)
	AuthMethod  string `json:"auth_method,omitempty"`    // oauth, token
	ConnectMode string `json:"connect_mode,omitempty"`   // stdio, grpc
	Workspace   string `json:"workspace,omitempty"`      // For CLI-based providers
}

// ModelConfig represents a model configuration that references a provider for connectivity.
// The model field uses protocol prefix format: [protocol/]model-identifier
// Supported protocols: openai, anthropic, antigravity, claude-cli, codex-cli, github-copilot
// Default protocol is "openai" if no prefix is specified.
type ModelConfig struct {
	// Required fields
	ModelName string `json:"model_name"`          // User-facing alias for the model
	Model     string `json:"model"`               // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")
	Provider  string `json:"provider,omitempty"`   // FK → ProviderConfig.Name

	// Optional optimizations
	RPM            int    `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
	RequestTimeout int    `json:"request_timeout,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"` // Extended thinking: off|low|medium|high|xhigh|adaptive
}

// Validate checks if the ModelConfig has all required fields.
func (c *ModelConfig) Validate() error {
	if c.ModelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

type GatewayConfig struct {
	Host string `json:"host" env:"OK_GATEWAY_HOST"`
	Port int    `json:"port" env:"OK_GATEWAY_PORT"`
}

type ToolConfig struct {
	Enabled bool `json:"enabled" env:"ENABLED"`
}

type BraveConfig struct {
	Enabled    bool   `json:"enabled"     env:"OK_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string `json:"api_key"     env:"OK_TOOLS_WEB_BRAVE_API_KEY"`
	MaxResults int    `json:"max_results" env:"OK_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

type TavilyConfig struct {
	Enabled    bool   `json:"enabled"     env:"OK_TOOLS_WEB_TAVILY_ENABLED"`
	APIKey     string `json:"api_key"     env:"OK_TOOLS_WEB_TAVILY_API_KEY"`
	BaseURL    string `json:"base_url"    env:"OK_TOOLS_WEB_TAVILY_BASE_URL"`
	MaxResults int    `json:"max_results" env:"OK_TOOLS_WEB_TAVILY_MAX_RESULTS"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"     env:"OK_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" env:"OK_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type PerplexityConfig struct {
	Enabled    bool   `json:"enabled"     env:"OK_TOOLS_WEB_PERPLEXITY_ENABLED"`
	APIKey     string `json:"api_key"     env:"OK_TOOLS_WEB_PERPLEXITY_API_KEY"`
	MaxResults int    `json:"max_results" env:"OK_TOOLS_WEB_PERPLEXITY_MAX_RESULTS"`
}

type SearXNGConfig struct {
	Enabled    bool   `json:"enabled"     env:"OK_TOOLS_WEB_SEARXNG_ENABLED"`
	BaseURL    string `json:"base_url"    env:"OK_TOOLS_WEB_SEARXNG_BASE_URL"`
	MaxResults int    `json:"max_results" env:"OK_TOOLS_WEB_SEARXNG_MAX_RESULTS"`
}

type GLMSearchConfig struct {
	Enabled bool   `json:"enabled"  env:"OK_TOOLS_WEB_GLM_ENABLED"`
	APIKey  string `json:"api_key"  env:"OK_TOOLS_WEB_GLM_API_KEY"`
	BaseURL string `json:"base_url" env:"OK_TOOLS_WEB_GLM_BASE_URL"`
	// SearchEngine specifies the search backend: "search_std" (default),
	// "search_pro", "search_pro_sogou", or "search_pro_quark".
	SearchEngine string `json:"search_engine" env:"OK_TOOLS_WEB_GLM_SEARCH_ENGINE"`
	MaxResults   int    `json:"max_results"   env:"OK_TOOLS_WEB_GLM_MAX_RESULTS"`
}

type WebToolsConfig struct {
	ToolConfig `                 envPrefix:"OK_TOOLS_WEB_"`
	Brave      BraveConfig      `                                json:"brave"`
	Tavily     TavilyConfig     `                                json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `                                json:"duckduckgo"`
	Perplexity PerplexityConfig `                                json:"perplexity"`
	SearXNG    SearXNGConfig    `                                json:"searxng"`
	GLMSearch       GLMSearchConfig `                                json:"glm_search"`
	FetchLimitBytes int64           `json:"fetch_limit_bytes,omitempty" env:"OK_TOOLS_WEB_FETCH_LIMIT_BYTES"`
}

type CronToolsConfig struct {
	ToolConfig         `    envPrefix:"OK_TOOLS_CRON_"`
	ExecTimeoutMinutes int `                                 env:"OK_TOOLS_CRON_EXEC_TIMEOUT_MINUTES" json:"exec_timeout_minutes"` // 0 means no timeout
}

type ExecConfig struct {
	ToolConfig          `         envPrefix:"OK_TOOLS_EXEC_"`
	EnableDenyPatterns  bool     `                                 env:"OK_TOOLS_EXEC_ENABLE_DENY_PATTERNS"  json:"enable_deny_patterns"`
	CustomDenyPatterns  []string `                                 env:"OK_TOOLS_EXEC_CUSTOM_DENY_PATTERNS"  json:"custom_deny_patterns"`
	CustomAllowPatterns []string `                                 env:"OK_TOOLS_EXEC_CUSTOM_ALLOW_PATTERNS" json:"custom_allow_patterns"`
	TimeoutSeconds      int      `                                 env:"OK_TOOLS_EXEC_TIMEOUT_SECONDS"       json:"timeout_seconds"` // 0 means use default (60s)
}

type SkillsToolsConfig struct {
	ToolConfig            `                       envPrefix:"OK_TOOLS_SKILLS_"`
	Registries            SkillsRegistriesConfig `                                   json:"registries"`
	MaxConcurrentSearches int                    `                                   json:"max_concurrent_searches" env:"OK_TOOLS_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig      `                                   json:"search_cache"`
}

type MediaCleanupConfig struct {
	ToolConfig `    envPrefix:"OK_MEDIA_CLEANUP_"`
	MaxAge     int `                                    env:"OK_MEDIA_CLEANUP_MAX_AGE"  json:"max_age_minutes"`
	Interval   int `                                    env:"OK_MEDIA_CLEANUP_INTERVAL" json:"interval_minutes"`
}

type ToolsConfig struct {
	AllowReadPaths  []string           `json:"allow_read_paths"  env:"OK_TOOLS_ALLOW_READ_PATHS"`
	AllowWritePaths []string           `json:"allow_write_paths" env:"OK_TOOLS_ALLOW_WRITE_PATHS"`
	Web             WebToolsConfig     `json:"web"`
	Cron            CronToolsConfig    `json:"cron"`
	Exec            ExecConfig         `json:"exec"`
	Skills          SkillsToolsConfig  `json:"skills"`
	MediaCleanup    MediaCleanupConfig `json:"media_cleanup"`
	AppendFile      ToolConfig         `json:"append_file"                                              envPrefix:"OK_TOOLS_APPEND_FILE_"`
	EditFile        ToolConfig         `json:"edit_file"                                                envPrefix:"OK_TOOLS_EDIT_FILE_"`
	FindSkills      ToolConfig         `json:"find_skills"                                              envPrefix:"OK_TOOLS_FIND_SKILLS_"`
	I2C             ToolConfig         `json:"i2c"                                                      envPrefix:"OK_TOOLS_I2C_"`
	InstallSkill    ToolConfig         `json:"install_skill"                                            envPrefix:"OK_TOOLS_INSTALL_SKILL_"`
	ListDir         ToolConfig         `json:"list_dir"                                                 envPrefix:"OK_TOOLS_LIST_DIR_"`
	Message         ToolConfig         `json:"message"                                                  envPrefix:"OK_TOOLS_MESSAGE_"`
	ReadFile        ToolConfig         `json:"read_file"                                                envPrefix:"OK_TOOLS_READ_FILE_"`
	SendFile        ToolConfig         `json:"send_file"                                                envPrefix:"OK_TOOLS_SEND_FILE_"`
	Spawn           ToolConfig         `json:"spawn"                                                    envPrefix:"OK_TOOLS_SPAWN_"`
	SPI             ToolConfig         `json:"spi"                                                      envPrefix:"OK_TOOLS_SPI_"`
	Subagent        ToolConfig         `json:"subagent"                                                 envPrefix:"OK_TOOLS_SUBAGENT_"`
	WebFetch        ToolConfig         `json:"web_fetch"                                                envPrefix:"OK_TOOLS_WEB_FETCH_"`
	WriteFile       ToolConfig         `json:"write_file"                                               envPrefix:"OK_TOOLS_WRITE_FILE_"`
}

type SearchCacheConfig struct {
	MaxSize    int `json:"max_size"    env:"OK_SKILLS_SEARCH_CACHE_MAX_SIZE"`
	TTLSeconds int `json:"ttl_seconds" env:"OK_SKILLS_SEARCH_CACHE_TTL_SECONDS"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `json:"enabled"           env:"OK_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL         string `json:"base_url"          env:"OK_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	AuthToken       string `json:"auth_token"        env:"OK_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"`
	SearchPath      string `json:"search_path"       env:"OK_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath      string `json:"skills_path"       env:"OK_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
	DownloadPath    string `json:"download_path"     env:"OK_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"`
	Timeout         int    `json:"timeout"           env:"OK_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"`
	MaxZipSize      int    `json:"max_zip_size"      env:"OK_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"`
	MaxResponseSize int    `json:"max_response_size" env:"OK_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE"`
}


func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.DebugCF("config", "Config file not found, using defaults", map[string]any{"path": path})
			return cfg, nil
		}
		logger.ErrorCF("config", "Config file read error", map[string]any{"path": path, "error": err.Error()})
		return nil, err
	}

	// Pre-scan the JSON to check whether the user provided model_list/provider_list entries.
	// Go's JSON decoder reuses existing slice backing-array elements rather than
	// zero-initializing them, so fields absent from the user's JSON (e.g. api_base)
	// would silently inherit values from the DefaultConfig template at the same
	// index position. We only reset slices when the user actually provides
	// entries; when count is 0 we keep DefaultConfig's built-in list as fallback.
	var tmp struct {
		ProviderList json.RawMessage `json:"provider_list"`
		ModelList    json.RawMessage `json:"model_list"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	if len(tmp.ProviderList) > 2 { // "[]" is 2 bytes
		cfg.ProviderList = nil
	}
	if len(tmp.ModelList) > 2 { // "[]" is 2 bytes
		cfg.ModelList = nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		logger.ErrorCF("config", "Config parse error", map[string]any{"path": path, "error": err.Error()})
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Migrate legacy channel config fields to new unified structures
	cfg.migrateChannelConfigs()

	// Validate model_list for uniqueness and required fields
	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	// Count enabled channels
	enabledChannels := 0
	if cfg.Channels.Telegram.Enabled {
		enabledChannels++
	}
	if cfg.Channels.Discord.Enabled {
		enabledChannels++
	}
	if cfg.Channels.WhatsApp.Enabled {
		enabledChannels++
	}
	if cfg.Channels.Slack.Enabled {
		enabledChannels++
	}

	logger.InfoCF("config", "Config loaded", map[string]any{
		"path":             path,
		"models":           len(cfg.ModelList),
		"agents":           len(cfg.Agents.List),
		"enabled_channels": enabledChannels,
		"debug":            cfg.Debug,
	})

	return cfg, nil
}

func (c *Config) migrateChannelConfigs() {
	// Discord: mention_only -> group_trigger.mention_only
	if c.Channels.Discord.MentionOnly && !c.Channels.Discord.GroupTrigger.MentionOnly {
		c.Channels.Discord.GroupTrigger.MentionOnly = true
	}

}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	if err := utils.WriteFileAtomic(path, data, 0o600); err != nil {
		return err
	}
	logger.DebugCF("config", "Config saved", map[string]any{"path": path})
	return nil
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Agents.Defaults.Workspace)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

// GetModelConfig returns the ModelConfig for the given model name.
// If multiple configs exist with the same model_name, it uses round-robin
// selection for load balancing. Returns an error if the model is not found.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	matches := c.findMatches(modelName)
	if len(matches) == 0 {
		return nil, fmt.Errorf("model %q not found in model_list", modelName)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	idx := rrCounter.Add(1) % uint64(len(matches))
	return &matches[idx], nil
}

// findMatches finds all ModelConfig entries with the given model_name.
func (c *Config) findMatches(modelName string) []ModelConfig {
	var matches []ModelConfig
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			matches = append(matches, c.ModelList[i])
		}
	}
	return matches
}

// ValidateModelList validates all ModelConfig entries in the model_list.
// It checks that each model config is valid.
// Note: Multiple entries with the same model_name are allowed for load balancing.
func (c *Config) ValidateModelList() error {
	for i := range c.ModelList {
		if err := c.ModelList[i].Validate(); err != nil {
			return fmt.Errorf("model_list[%d]: %w", i, err)
		}
	}
	return nil
}

// GetProviderConfig returns the ProviderConfig with the given name, or nil if not found.
func (c *Config) GetProviderConfig(name string) *ProviderConfig {
	for i := range c.ProviderList {
		if c.ProviderList[i].Name == name {
			return &c.ProviderList[i]
		}
	}
	return nil
}

// ResolveModelProvider finds the ProviderConfig for a model.
// It looks up m.Provider in ProviderList first, then falls back to deriving
// from the protocol prefix of m.Model (backward compat).
func (c *Config) ResolveModelProvider(m *ModelConfig) (*ProviderConfig, error) {
	if m.Provider != "" {
		if p := c.GetProviderConfig(m.Provider); p != nil {
			return p, nil
		}
		return nil, fmt.Errorf("provider %q not found in provider_list (referenced by model %q)", m.Provider, m.ModelName)
	}

	// Derive from model protocol prefix
	protocol := "openai"
	if prefix, _, found := strings.Cut(m.Model, "/"); found {
		protocol = prefix
	}
	if p := c.GetProviderConfig(protocol); p != nil {
		return p, nil
	}

	// Return a synthetic provider with just the name so the factory can use defaults
	return &ProviderConfig{Name: protocol}, nil
}

func (t *ToolsConfig) IsToolEnabled(name string) bool {
	switch name {
	case "web":
		return t.Web.Enabled
	case "cron":
		return t.Cron.Enabled
	case "exec":
		return t.Exec.Enabled
	case "skills":
		return t.Skills.Enabled
	case "media_cleanup":
		return t.MediaCleanup.Enabled
	case "append_file":
		return t.AppendFile.Enabled
	case "edit_file":
		return t.EditFile.Enabled
	case "find_skills":
		return t.FindSkills.Enabled
	case "i2c":
		return t.I2C.Enabled
	case "install_skill":
		return t.InstallSkill.Enabled
	case "list_dir":
		return t.ListDir.Enabled
	case "message":
		return t.Message.Enabled
	case "read_file":
		return t.ReadFile.Enabled
	case "spawn":
		return t.Spawn.Enabled
	case "spi":
		return t.SPI.Enabled
	case "subagent":
		return t.Subagent.Enabled
	case "web_fetch":
		return t.WebFetch.Enabled
	case "send_file":
		return t.SendFile.Enabled
	case "write_file":
		return t.WriteFile.Enabled
	default:
		return true
	}
}
