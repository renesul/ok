// OK - Lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 OK contributors

package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	channels "ok/app/input"
	events "ok/app/input/bus"
	"ok/app/memory"
	"ok/app/output"
	"ok/app/routing"
	"ok/app/types"
	"ok/internal/auth"
	"ok/internal/commands"
	"ok/internal/config"
	"ok/internal/logger"
	"ok/internal/mcp"
	"ok/internal/media"
	"ok/internal/skills"
	"ok/internal/utils"
	"ok/internal/voice"
	"ok/providers"
	tools "ok/app/execution"
)

type AgentLoop struct {
	bus            *events.MessageBus
	cfg            *config.Config
	registry       *AgentRegistry
	state          *utils.Manager
	running        atomic.Bool
	fallback       *providers.FallbackChain
	channelManager *channels.Manager
	mediaStore     media.MediaStore
	transcriber    voice.Transcriber
	cmdRegistry    *commands.Registry
	planner        *Planner
	toolExec       *ToolExecutor
	memory         *MemoryManager
	summarizer     *output.Summarizer
	mcpManager     *mcp.Manager
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string   // Session identifier for history/context
	Channel         string   // Target channel for tool execution
	ChatID          string   // Target chat ID for tool execution
	UserMessage     string   // User message content (may include prefix)
	Media           []string // media:// refs from inbound message
	DefaultResponse string   // Response when LLM returns empty
	EnableSummary   bool     // Whether to trigger summarization
	SendResponse    bool     // Whether to send response via bus
	NoHistory       bool     // If true, don't load session history (for heartbeat)
}

const (
	defaultResponse           = "I've completed processing but have no response to give. Increase `max_tool_iterations` in config.json."
	sessionKeyAgentPrefix     = "agent:"
	metadataKeyAccountID = "account_id"
)

func NewAgentLoop(
	cfg *config.Config,
	msgBus *events.MessageBus,
	provider providers.LLMProvider,
) *AgentLoop {
	registry := NewAgentRegistry(cfg, provider)

	// Register shared tools to all agents
	registerSharedTools(cfg, msgBus, registry, provider)

	// Set up shared fallback chain
	cooldown := providers.NewCooldownTracker()
	fallbackChain := providers.NewFallbackChain(cooldown)

	// Create state manager using default agent's workspace for channel recording
	defaultAgent := registry.GetDefaultAgent()
	var stateManager *utils.Manager
	if defaultAgent != nil {
		stateManager = utils.NewManager(defaultAgent.Workspace)
	}

	toolExec := &ToolExecutor{
		bus: msgBus,
	}
	memMgr := &MemoryManager{
		maxMediaSize: cfg.Agents.Defaults.GetMaxMediaSize(),
	}
	planner := newPlanner(fallbackChain, msgBus, toolExec, memMgr)

	// Initialize RAG if configured
	ragBaseURL := cfg.RAG.BaseURL
	if ragBaseURL == "" && len(cfg.ModelList) > 0 {
		ragBaseURL = cfg.ModelList[0].APIBase
	}
	if cfg.RAG.Enabled && ragBaseURL != "" && cfg.RAG.Model != "" {
		if defaultAgent != nil {
			ragDir := filepath.Join(defaultAgent.Workspace, "rag")
			store := memory.NewInteractionStore(ragDir)
			if err := store.Open(); err != nil {
				logger.ErrorCF("rag", "Failed to open RAG store", map[string]any{"error": err.Error()})
			} else {
				ragAPIKey := cfg.RAG.APIKey
				if ragAPIKey == "" {
					ragAPIKey = resolveAPIKeyFromAuthStore(ragBaseURL)
				}
				embedder := memory.NewHTTPEmbedder(ragBaseURL, ragAPIKey, cfg.RAG.Model)
				store.SetEmbeddingModel(cfg.RAG.Model)
				retriever := memory.NewRetriever(store, embedder, cfg.RAG.TopK, cfg.RAG.MinSimilarity)
				memMgr.retriever = retriever
				defaultAgent.ContextBuilder.SetRetriever(retriever)
				logger.InfoCF("rag", "RAG initialized", map[string]any{
					"model":     cfg.RAG.Model,
					"store_dir": ragDir,
					"count":     store.Count(),
				})
			}
		}
	}

	// Initialize MCP servers
	var mcpMgr *mcp.Manager
	if len(cfg.MCPServers) > 0 {
		mcpMgr = mcp.NewManager()
		startCtx, startCancel := context.WithTimeout(context.Background(), 60*time.Second)
		mcpMgr.LoadFromConfig(startCtx, cfg.MCPServers)
		startCancel()

		// Register discovered tools with all agents
		for serverName, serverTools := range mcpMgr.GetAllTools() {
			serverCfg, _ := mcpMgr.GetServerConfig(serverName)
			for _, mcpTool := range serverTools {
				wrappedTool := tools.NewMCPTool(mcpMgr, serverName, serverCfg.ToolPrefix, mcpTool)
				for _, agentID := range registry.ListAgentIDs() {
					if agent, ok := registry.GetAgent(agentID); ok {
						agent.Tools.Register(wrappedTool)
					}
				}
			}
		}
	}

	al := &AgentLoop{
		bus:         msgBus,
		cfg:         cfg,
		registry:    registry,
		state:       stateManager,
		fallback:    fallbackChain,
		cmdRegistry: commands.NewRegistry(commands.BuiltinDefinitions()),
		planner:     planner,
		toolExec:    toolExec,
		memory:      memMgr,
		summarizer:  output.NewSummarizer(),
		mcpManager:  mcpMgr,
	}

	return al
}

// registerSharedTools registers tools that are shared across all agents (web, message, spawn).
func registerSharedTools(
	cfg *config.Config,
	msgBus *events.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
) {
	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		// Web tools
		if cfg.Tools.IsToolEnabled("web") {
			searchTool, err := tools.NewWebSearchTool(tools.WebSearchToolOptions{
				BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
				BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
				BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
				TavilyAPIKey:         cfg.Tools.Web.Tavily.APIKey,
				TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
				TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
				TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
				DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
				DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
				PerplexityAPIKey:     cfg.Tools.Web.Perplexity.APIKey,
				PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
				PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
				SearXNGBaseURL:       cfg.Tools.Web.SearXNG.BaseURL,
				SearXNGMaxResults:    cfg.Tools.Web.SearXNG.MaxResults,
				SearXNGEnabled:       cfg.Tools.Web.SearXNG.Enabled,
				GLMSearchAPIKey:      cfg.Tools.Web.GLMSearch.APIKey,
				GLMSearchBaseURL:     cfg.Tools.Web.GLMSearch.BaseURL,
				GLMSearchEngine:      cfg.Tools.Web.GLMSearch.SearchEngine,
				GLMSearchMaxResults:  cfg.Tools.Web.GLMSearch.MaxResults,
				GLMSearchEnabled:     cfg.Tools.Web.GLMSearch.Enabled,
				Proxy:                cfg.Proxy,
			})
			if err != nil {
				logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{"error": err.Error()})
			} else if searchTool != nil {
				agent.Tools.Register(searchTool)
			}
		}
		if cfg.Tools.IsToolEnabled("web_fetch") {
			fetchTool, err := tools.NewWebFetchToolWithProxy(50000, cfg.Proxy, cfg.Tools.Web.FetchLimitBytes)
			if err != nil {
				logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
			} else {
				agent.Tools.Register(fetchTool)
			}
		}

		// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
		if cfg.Tools.IsToolEnabled("i2c") {
			agent.Tools.Register(tools.NewI2CTool())
		}
		if cfg.Tools.IsToolEnabled("spi") {
			agent.Tools.Register(tools.NewSPITool())
		}

		// Message tool
		if cfg.Tools.IsToolEnabled("message") {
			messageTool := tools.NewMessageTool()
			messageTool.SetSendCallback(func(channel, chatID, content string) error {
				pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pubCancel()
				return msgBus.PublishOutbound(pubCtx, events.OutboundMessage{
					Channel: channel,
					ChatID:  chatID,
					Content: content,
				})
			})
			agent.Tools.Register(messageTool)
		}

		// Send file tool (outbound media via MediaStore — store injected later by SetMediaStore)
		if cfg.Tools.IsToolEnabled("send_file") {
			sendFileTool := tools.NewSendFileTool(
				agent.Workspace,
				cfg.Agents.Defaults.RestrictToWorkspace,
				cfg.Agents.Defaults.GetMaxMediaSize(),
				nil,
			)
			agent.Tools.Register(sendFileTool)
		}

		// Skill discovery and installation tools
		skills_enabled := cfg.Tools.IsToolEnabled("skills")
		find_skills_enable := cfg.Tools.IsToolEnabled("find_skills")
		install_skills_enable := cfg.Tools.IsToolEnabled("install_skill")
		if skills_enabled && (find_skills_enable || install_skills_enable) {
			registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
				MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
				ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
			})

			if find_skills_enable {
				searchCache := skills.NewSearchCache(
					cfg.Tools.Skills.SearchCache.MaxSize,
					time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
				)
				agent.Tools.Register(tools.NewFindSkillsTool(registryMgr, searchCache))
			}

			if install_skills_enable {
				agent.Tools.Register(tools.NewInstallSkillTool(registryMgr, agent.Workspace))
			}
		}

		// Spawn tool with allowlist checker
		if cfg.Tools.IsToolEnabled("spawn") {
			if cfg.Tools.IsToolEnabled("subagent") {
				subagentManager := tools.NewSubagentManager(provider, agent.Model, agent.Workspace)
				subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)
				spawnTool := tools.NewSpawnTool(subagentManager)
				currentAgentID := agentID
				spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
					return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
				})
				agent.Tools.Register(spawnTool)
			} else {
				logger.WarnCF("agent", "spawn tool requires subagent to be enabled", nil)
			}
		}
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			// Process message
			func() {
				// TODO: Re-enable media cleanup after inbound media is properly consumed by the agent.
				// Currently disabled because files are deleted before the LLM can access their content.
				// defer func() {
				// 	if al.mediaStore != nil && msg.MediaScope != "" {
				// 		if releaseErr := al.mediaStore.ReleaseAll(msg.MediaScope); releaseErr != nil {
				// 			logger.WarnCF("agent", "Failed to release media", map[string]any{
				// 				"scope": msg.MediaScope,
				// 				"error": releaseErr.Error(),
				// 			})
				// 		}
				// 	}
				// }()

				response, err := al.processMessage(ctx, msg)
				if err != nil {
					response = fmt.Sprintf("Error processing message: %v", err)
				}

				if response != "" {
					// Check if the message tool already sent a response during this round.
					// If so, skip publishing to avoid duplicate messages to the user.
					// Use default agent's tools to check (message tool is shared).
					alreadySent := false
					defaultAgent := al.registry.GetDefaultAgent()
					if defaultAgent != nil {
						if tool, ok := defaultAgent.Tools.Get("message"); ok {
							if mt, ok := tool.(*tools.MessageTool); ok {
								alreadySent = mt.HasSentInRound()
							}
						}
					}

					if !alreadySent {
						al.bus.PublishOutbound(ctx, events.OutboundMessage{
							Channel: msg.Channel,
							ChatID:  msg.ChatID,
							Content: response,
						})
						logger.InfoCF("agent", "Published outbound response",
							map[string]any{
								"channel":     msg.Channel,
								"chat_id":     msg.ChatID,
								"content_len": len(response),
							})
					} else {
						logger.DebugCF(
							"agent",
							"Skipped outbound (message tool already sent)",
							map[string]any{"channel": msg.Channel},
						)
					}
				}
			}()
		}
	}

	return nil
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
	if al.mcpManager != nil {
		al.mcpManager.Close()
	}
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	for _, agentID := range al.registry.ListAgentIDs() {
		if agent, ok := al.registry.GetAgent(agentID); ok {
			agent.Tools.Register(tool)
		}
	}
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
	al.planner.setChannelManager(cm)
}

// GetSessionHistory resolves the session key for a channel/chatID pair
// and returns the stored conversation history. Used by the chat channel
// to restore messages on WebSocket reconnect.
func (al *AgentLoop) GetSessionHistory(channel, chatID string) []providers.Message {
	route := al.registry.ResolveRoute(types.RouteInput{
		Channel: channel,
		Peer:    &types.RoutePeer{Kind: "direct", ID: chatID},
	})
	agent, ok := al.registry.GetAgent(route.AgentID)
	if !ok {
		agent = al.registry.GetDefaultAgent()
	}
	if agent == nil {
		return nil
	}
	return agent.Sessions.GetHistory(route.SessionKey)
}

// SetMediaStore injects a MediaStore for media lifecycle management.
func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s
	al.memory.mediaStore = s
	al.toolExec.mediaStore = s

	// Propagate store to send_file tools in all agents.
	al.registry.ForEachTool("send_file", func(t tools.Tool) {
		if sf, ok := t.(*tools.SendFileTool); ok {
			sf.SetMediaStore(s)
		}
	})
}

// SetTranscriber injects a voice transcriber for agent-level audio transcription.
func (al *AgentLoop) SetTranscriber(t voice.Transcriber) {
	al.transcriber = t
}

var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)

// transcribeAudioInMessage resolves audio media refs, transcribes them, and
// replaces audio annotations in msg.Content with the transcribed text.
func (al *AgentLoop) transcribeAudioInMessage(ctx context.Context, msg events.InboundMessage) events.InboundMessage {
	if al.transcriber == nil || al.mediaStore == nil || len(msg.Media) == 0 {
		return msg
	}

	// Transcribe each audio media ref in order.
	var transcriptions []string
	for _, ref := range msg.Media {
		path, meta, err := al.mediaStore.ResolveWithMeta(ref)
		if err != nil {
			logger.WarnCF("voice", "Failed to resolve media ref", map[string]any{"ref": ref, "error": err})
			continue
		}
		if !utils.IsAudioFile(meta.Filename, meta.ContentType) {
			continue
		}
		result, err := al.transcriber.Transcribe(ctx, path)
		if err != nil {
			logger.WarnCF("voice", "Transcription failed", map[string]any{"ref": ref, "error": err})
			transcriptions = append(transcriptions, "")
			continue
		}
		transcriptions = append(transcriptions, result.Text)
	}

	if len(transcriptions) == 0 {
		return msg
	}

	// Replace audio annotations sequentially with transcriptions.
	idx := 0
	newContent := audioAnnotationRe.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if idx >= len(transcriptions) {
			return match
		}
		text := transcriptions[idx]
		idx++
		return "[voice: " + text + "]"
	})

	// Append any remaining transcriptions not matched by an annotation.
	for ; idx < len(transcriptions); idx++ {
		newContent += "\n[voice: " + transcriptions[idx] + "]"
	}

	msg.Content = newContent
	return msg
}

// inferMediaType determines the media type ("image", "audio", "video", "file")
// from a filename and MIME content type.
func inferMediaType(filename, contentType string) string {
	ct := strings.ToLower(contentType)
	fn := strings.ToLower(filename)

	if strings.HasPrefix(ct, "image/") {
		return "image"
	}
	if strings.HasPrefix(ct, "audio/") || ct == "application/ogg" {
		return "audio"
	}
	if strings.HasPrefix(ct, "video/") {
		return "video"
	}

	// Fallback: infer from extension
	ext := filepath.Ext(fn)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	}

	return "file"
}

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChannel(channel string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChatID(chatID)
}

func (al *AgentLoop) ProcessDirect(
	ctx context.Context,
	content, sessionKey string,
) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

func (al *AgentLoop) ProcessDirectWithChannel(
	ctx context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	msg := events.InboundMessage{
		Channel:    channel,
		SenderID:   "cron",
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(
	ctx context.Context,
	content, channel, chatID string,
) (string, error) {
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for heartbeat")
	}
	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true, // Don't load session history for heartbeat
	})
}

func (al *AgentLoop) processMessage(ctx context.Context, msg events.InboundMessage) (string, error) {
	// Add message preview to log (show full content for error messages)
	var logContent string
	if strings.Contains(msg.Content, "Error:") || strings.Contains(msg.Content, "error") {
		logContent = msg.Content // Full content for errors
	} else {
		logContent = utils.Truncate(msg.Content, 80)
	}
	logger.InfoCF(
		"agent",
		fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]any{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		},
	)

	msg = al.transcribeAudioInMessage(ctx, msg)

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	route, agent, routeErr := al.resolveMessageRoute(msg)

	// Commands are checked before requiring a successful route.
	// Global commands (/help, /show, /switch) work even when routing fails;
	// context-dependent commands check their own Runtime fields and report
	// "unavailable" when the required capability is nil.
	if response, handled := al.handleCommand(ctx, msg, agent); handled {
		return response, nil
	}

	if routeErr != nil {
		return "", routeErr
	}

	// Reset message-tool state for this round so we don't skip publishing due to a previous round.
	if tool, ok := agent.Tools.Get("message"); ok {
		if resetter, ok := tool.(interface{ ResetSentInRound() }); ok {
			resetter.ResetSentInRound()
		}
	}

	// Resolve session key from route, while preserving explicit agent-scoped keys.
	scopeKey := resolveScopeKey(route, msg.SessionKey)
	sessionKey := scopeKey

	logger.InfoCF("agent", "Routed message",
		map[string]any{
			"agent_id":      agent.ID,
			"scope_key":     scopeKey,
			"session_key":   sessionKey,
			"matched_by":    route.MatchedBy,
			"route_agent":   route.AgentID,
			"route_channel": route.Channel,
		})

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		Media:           msg.Media,
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
		SendResponse:    false,
	})
}

func (al *AgentLoop) resolveMessageRoute(msg events.InboundMessage) (types.ResolvedRoute, *AgentInstance, error) {
	route := al.registry.ResolveRoute(types.RouteInput{
		Channel:   msg.Channel,
		AccountID: inboundMetadata(msg, metadataKeyAccountID),
		Peer:      extractPeer(msg),
	})

	agent, ok := al.registry.GetAgent(route.AgentID)
	if !ok {
		agent = al.registry.GetDefaultAgent()
	}
	if agent == nil {
		return types.ResolvedRoute{}, nil, fmt.Errorf("no agent available for route (agent_id=%s)", route.AgentID)
	}

	return route, agent, nil
}

func resolveScopeKey(route types.ResolvedRoute, msgSessionKey string) string {
	if msgSessionKey != "" && strings.HasPrefix(msgSessionKey, sessionKeyAgentPrefix) {
		return msgSessionKey
	}
	return route.SessionKey
}

func (al *AgentLoop) processSystemMessage(
	ctx context.Context,
	msg events.InboundMessage,
) (string, error) {
	if msg.Channel != "system" {
		return "", fmt.Errorf(
			"processSystemMessage called with non-system message channel: %s",
			msg.Channel,
		)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]any{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel, originChatID string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
		originChatID = msg.ChatID[idx+1:]
	} else {
		originChannel = "cli"
		originChatID = msg.ChatID
	}

	// Extract subagent result from message content
	// Format: "Task 'label' completed.\n\nResult:\n<actual content>"
	content := msg.Content
	if idx := strings.Index(content, "Result:\n"); idx >= 0 {
		content = content[idx+8:] // Extract just the result part
	}

	// Skip internal channels - only log, don't send to user
	if utils.IsInternalChannel(originChannel) {
		logger.InfoCF("agent", "Subagent completed (internal channel)",
			map[string]any{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
			})
		return "", nil
	}

	// Use default agent for system messages
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for system message")
	}

	// Use the origin session for context
	sessionKey := routing.BuildAgentMainSessionKey(agent.ID)

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         originChannel,
		ChatID:          originChatID,
		UserMessage:     fmt.Sprintf("[System: %s] %s", msg.SenderID, msg.Content),
		DefaultResponse: "Background task completed.",
		EnableSummary:   false,
		SendResponse:    true,
	})
}

// runAgentLoop is the core message processing logic.
func (al *AgentLoop) runAgentLoop(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
) (string, error) {
	// 0. Record last channel for heartbeat notifications (skip internal channels and cli)
	if opts.Channel != "" && opts.ChatID != "" {
		if !utils.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF(
					"agent",
					"Failed to record last channel",
					map[string]any{"error": err.Error()},
				)
			}
		}
	}

	// 1. Load memory (build messages)
	convCtx := al.memory.Load(agent, MemoryLoadOptions{
		SessionKey:  opts.SessionKey,
		UserMessage: opts.UserMessage,
		Media:       opts.Media,
		Channel:     opts.Channel,
		ChatID:      opts.ChatID,
		NoHistory:   opts.NoHistory,
	})

	// 2. Save user message to session
	al.memory.SaveUserMessage(agent, opts.SessionKey, opts.UserMessage)

	// 3. Plan (LLM loop)
	result, err := al.planner.Run(ctx, agent, convCtx.Messages, PlanOptions{
		SessionKey:   opts.SessionKey,
		Channel:      opts.Channel,
		ChatID:       opts.ChatID,
		UserMessage:  opts.UserMessage,
		SendResponse: opts.SendResponse,
	})
	if err != nil {
		return "", err
	}

	// 4. Handle empty response
	finalContent := result.Content
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 5. Save assistant message
	al.memory.SaveAssistantMessage(agent, opts.SessionKey, finalContent)

	// 6. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 7. Optional: send response via bus
	if opts.SendResponse {
		al.bus.PublishOutbound(ctx, events.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalContent,
		})
	}

	// 8. Log response
	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]any{
			"agent_id":     agent.ID,
			"session_key":  opts.SessionKey,
			"iterations":   result.Iterations,
			"final_length": len(finalContent),
		})

	return finalContent, nil
}

// targetReasoningChannelID delegates to planner.
func (al *AgentLoop) targetReasoningChannelID(channelName string) string {
	return al.planner.targetReasoningChannelID(channelName)
}

// handleReasoning delegates to planner.
func (al *AgentLoop) handleReasoning(
	ctx context.Context,
	reasoningContent, channelName, channelID string,
) {
	al.planner.handleReasoning(ctx, reasoningContent, channelName, channelID)
}


// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	al.summarizer.MaybeSummarize(agent, agent.Sessions, sessionKey)
}


// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return info
	}

	// Tools info
	toolsList := agent.Tools.List()
	info["tools"] = map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(al.registry.ListAgentIDs()),
		"ids":   al.registry.ListAgentIDs(),
	}

	return info
}



func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg events.InboundMessage,
	agent *AgentInstance,
) (string, bool) {
	if !commands.HasCommandPrefix(msg.Content) {
		return "", false
	}

	if al.cmdRegistry == nil {
		return "", false
	}

	rt := al.buildCommandsRuntime(agent)
	executor := commands.NewExecutor(al.cmdRegistry, rt)

	var commandReply string
	result := executor.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     msg.Content,
		Reply: func(text string) error {
			commandReply = text
			return nil
		},
	})

	switch result.Outcome {
	case commands.OutcomeHandled:
		if result.Err != nil {
			return mapCommandError(result), true
		}
		if commandReply != "" {
			return commandReply, true
		}
		return "", true
	default: // OutcomePassthrough — let the message fall through to LLM
		return "", false
	}
}

func (al *AgentLoop) buildCommandsRuntime(agent *AgentInstance) *commands.Runtime {
	rt := &commands.Runtime{
		Config:          al.cfg,
		ListAgentIDs:    al.registry.ListAgentIDs,
		ListDefinitions: al.cmdRegistry.Definitions,
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel '%s' not found or not enabled", value)
			}
			return nil
		},
	}
	if agent != nil {
		rt.GetModelInfo = func() (string, string) {
			agent.modelMu.RLock()
			defer agent.modelMu.RUnlock()
			return agent.Model, al.cfg.Agents.Defaults.Provider
		}
		rt.SwitchModel = func(value string) (string, error) {
			agent.modelMu.Lock()
			defer agent.modelMu.Unlock()
			oldModel := agent.Model
			agent.Model = value
			return oldModel, nil
		}
	}
	return rt
}

func mapCommandError(result commands.ExecuteResult) string {
	if result.Command == "" {
		return fmt.Sprintf("Failed to execute command: %v", result.Err)
	}
	return fmt.Sprintf("Failed to execute /%s: %v", result.Command, result.Err)
}

// extractPeer extracts the routing peer from the inbound message's structured Peer field.
func extractPeer(msg events.InboundMessage) *types.RoutePeer {
	if msg.Peer.Kind == "" {
		return nil
	}
	peerID := msg.Peer.ID
	if peerID == "" {
		if msg.Peer.Kind == "direct" {
			peerID = msg.SenderID
		} else {
			peerID = msg.ChatID
		}
	}
	return &types.RoutePeer{Kind: msg.Peer.Kind, ID: peerID}
}

func inboundMetadata(msg events.InboundMessage, key string) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata[key]
}

// resolveAPIKeyFromAuthStore detects the provider from a base URL and returns
// the access token from the auth store. This allows RAG (and other features)
// to use credentials from connected providers without explicit api_key config.
func resolveAPIKeyFromAuthStore(baseURL string) string {
	providerMap := map[string][]string{
		"openai":    {"api.openai.com"},
		"anthropic": {"api.anthropic.com"},
		"groq":      {"api.groq.com"},
		"deepseek":  {"api.deepseek.com"},
		"mistral":   {"api.mistral.ai"},
		"xai":       {"api.x.ai"},
	}
	for provider, hosts := range providerMap {
		for _, host := range hosts {
			if strings.Contains(baseURL, host) {
				cred, err := auth.GetCredential(provider)
				if err == nil && cred != nil && cred.AccessToken != "" {
					return cred.AccessToken
				}
			}
		}
	}
	// Try custom providers
	store, err := auth.LoadStore()
	if err == nil {
		for name, cred := range store.Credentials {
			if strings.HasPrefix(name, "custom-") && cred.APIBase != "" && strings.Contains(baseURL, cred.APIBase) {
				return cred.AccessToken
			}
		}
	}
	return ""
}

