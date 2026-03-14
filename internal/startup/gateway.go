package startup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"ok/internal/appinfo"
	agent "ok/app/orchestrator"
	events "ok/app/input/bus"
	channels "ok/app/input"
	okchat "ok/app/input/chat"
	_ "ok/app/input/discord"
	_ "ok/app/input/slack"
	_ "ok/app/input/telegram"
	_ "ok/app/input/whatsapp"
	"ok/internal/config"
	"ok/internal/cron"
	"ok/internal/devices"
	"ok/internal/health"
	"ok/internal/heartbeat"
	"ok/internal/logger"
	"ok/internal/webui"
	"ok/internal/media"
	"ok/providers"
	"ok/internal/utils"
	tools "ok/app/execution"
	"ok/internal/voice"
)

// GatewayCmd starts the gateway with an optional debug flag.
func GatewayCmd(debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	// Initialize file-based logging
	logsDir := filepath.Join(appinfo.GetOKHome(), "logs")
	if err := logger.InitFileLogging(logsDir); err != nil {
		fmt.Printf("Warning: file logging unavailable: %v\n", err)
	} else {
		defer logger.CloseFileLogging()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	for {
		reason, err := runGateway(debug, sigChan)
		if err != nil {
			return err
		}
		if reason == shutdownSignal {
			return nil
		}
		// reason == shutdownReload — loop and restart
		fmt.Println("\n♻ Restarting gateway with new config...")
	}
}

type shutdownReason int

const (
	shutdownSignal shutdownReason = iota // Ctrl+C
	shutdownReload                       // triggered via API
)

// reloadChan is a package-level channel that the web UI triggers to reload the gateway.
var reloadChan = make(chan struct{}, 1)

// TriggerReload signals the gateway to reload its configuration.
func TriggerReload() {
	select {
	case reloadChan <- struct{}{}:
	default:
	}
}

func runGateway(debug bool, sigChan <-chan os.Signal) (shutdownReason, error) {
	cfg, err := appinfo.LoadConfig()
	if err != nil {
		return shutdownSignal, fmt.Errorf("error loading config: %w", err)
	}

	// Enable debug from config if not already enabled via CLI flag
	if cfg.Debug && !debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("Debug mode enabled (from config)")
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	setupMode := false
	if err != nil {
		// Setup mode: no provider configured yet — start with WebUI only
		logger.WarnCF("gateway", "Provider not available, starting in setup mode", map[string]any{
			"error": err.Error(),
		})
		fmt.Println("⚠ No providers configured — starting in setup mode")
		fmt.Println("  Configure via WebUI and reload to activate.")
		setupMode = true
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := events.NewMessageBus()

	var agentLoop *agent.AgentLoop
	var cronService *cron.CronService
	var heartbeatService *heartbeat.HeartbeatService

	if !setupMode {
		agentLoop = agent.NewAgentLoop(cfg, msgBus, provider)
		agentLoop.RegisterIntegrationTools()

		// Print agent startup info
		fmt.Println("\n📦 Agent Status:")
		startupInfo := agentLoop.GetStartupInfo()
		toolsInfo, _ := startupInfo["tools"].(map[string]any)
		skillsInfo, _ := startupInfo["skills"].(map[string]any)
		if toolsInfo == nil {
			toolsInfo = map[string]any{"count": 0}
		}
		if skillsInfo == nil {
			skillsInfo = map[string]any{"total": 0, "available": 0}
		}
		fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
		fmt.Printf("  • Skills: %d/%d available\n",
			skillsInfo["available"],
			skillsInfo["total"])

		// Log to file as well
		logger.InfoCF("agent", "Agent initialized",
			map[string]any{
				"tools_count":      toolsInfo["count"],
				"skills_total":     skillsInfo["total"],
				"skills_available": skillsInfo["available"],
			})

		// Setup cron tool and service
		execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
		var cronErr error
		cronService, cronErr = setupCronTool(
			agentLoop,
			msgBus,
			cfg.WorkspacePath(),
			cfg.Agents.Defaults.RestrictToWorkspace,
			execTimeout,
			cfg,
		)
		if cronErr != nil {
			return shutdownSignal, fmt.Errorf("cron setup failed: %w", cronErr)
		}

		heartbeatService = heartbeat.NewHeartbeatService(
			cfg.WorkspacePath(),
			cfg.Heartbeat.Interval,
			cfg.Heartbeat.Enabled,
		)
		heartbeatService.SetBus(msgBus)
		heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
			// Use cli:direct as fallback if no valid channel
			if channel == "" || chatID == "" {
				channel, chatID = "cli", "direct"
			}
			// Use ProcessHeartbeat - no session history, each heartbeat is independent
			var response string
			response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
			if err != nil {
				return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
			}
			if response == "HEARTBEAT_OK" {
				return tools.SilentResult("Heartbeat OK")
			}
			// For heartbeat, always return silent - the subagent result will be
			// sent to user via processSystemMessage when the async task completes
			return tools.SilentResult(response)
		})
	}

	// In setup mode, force WebUI and Chat channel so user can configure
	if setupMode {
		cfg.WebUI.Enabled = true
		cfg.Channels.Chat.Enabled = true
	}

	// Create media store for file lifecycle management with TTL cleanup
	mediaStore := media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	mediaStore.Start()

	channelManager, err := channels.NewManager(cfg, msgBus, mediaStore)
	if err != nil {
		mediaStore.Stop()
		return shutdownSignal, fmt.Errorf("error creating channel manager: %w", err)
	}

	if agentLoop != nil {
		// Inject channel manager and media store into agent loop
		agentLoop.SetChannelManager(channelManager)
		agentLoop.SetMediaStore(mediaStore)

		// Wire chat history restoration so the web channel sends past messages on connect.
		if ch, ok := channelManager.GetChannel("chat"); ok {
			if hc, ok := ch.(okchat.HistoryCapable); ok {
				hc.SetHistoryProvider(agentLoop.GetSessionHistory)
			}
		}

		// Wire up voice transcription if a supported provider is configured.
		if transcriber := voice.DetectTranscriber(cfg); transcriber != nil {
			agentLoop.SetTranscriber(transcriber)
			logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
		}
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cronService != nil {
		if err := cronService.Start(); err != nil {
			fmt.Printf("Error starting cron service: %v\n", err)
		}
		fmt.Println("✓ Cron service started")
	}

	if heartbeatService != nil {
		if err := heartbeatService.Start(); err != nil {
			fmt.Printf("Error starting heartbeat service: %v\n", err)
		}
		fmt.Println("✓ Heartbeat service started")
	}

	stateManager := utils.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	// Setup shared HTTP server with health endpoints and webhook handlers
	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	channelManager.SetupHTTPServer(addr, healthServer)

	// Start embedded web UI if enabled
	if cfg.WebUI.Enabled {
		webui.SetReloadFunc(TriggerReload)
		webui.Start(cfg.WebUI, appinfo.GetConfigPath())
	}

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
		return shutdownSignal, err
	}

	fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Printf("✓ Chat available at http://%s:%d/\n", cfg.Gateway.Host, cfg.Gateway.Port)

	if agentLoop != nil {
		go agentLoop.Run(ctx)
	}

	// Wait for Ctrl+C or reload signal from web UI
	var reason shutdownReason
	select {
	case <-sigChan:
		reason = shutdownSignal
	case <-reloadChan:
		reason = shutdownReload
		fmt.Println("\n⚡ Reload triggered via API, restarting...")
		logger.InfoC("gateway", "Reload triggered via web UI")
	}

	// Graceful shutdown
	if reason == shutdownSignal {
		fmt.Println("\nShutting down...")
	}
	if provider != nil {
		if cp, ok := provider.(providers.StatefulProvider); ok {
			cp.Close()
		}
	}
	cancel()
	msgBus.Close()

	// Use a fresh context with timeout for graceful shutdown,
	// since the original ctx is already canceled.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	channelManager.StopAll(shutdownCtx)
	deviceService.Stop()
	if heartbeatService != nil {
		heartbeatService.Stop()
	}
	if cronService != nil {
		cronService.Stop()
	}
	mediaStore.Stop()
	if agentLoop != nil {
		agentLoop.Stop()
	}

	if reason == shutdownSignal {
		fmt.Println("✓ Gateway stopped")
	} else {
		fmt.Println("✓ Gateway stopped for reload")
	}

	return reason, nil
}

func setupCronTool(
	agentLoop *agent.AgentLoop,
	msgBus *events.MessageBus,
	workspace string,
	restrict bool,
	execTimeout time.Duration,
	cfg *config.Config,
) (*cron.CronService, error) {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool if enabled
	var cronTool *tools.CronTool
	if cfg.Tools.IsToolEnabled("cron") {
		var err error
		cronTool, err = tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
		if err != nil {
			return nil, fmt.Errorf("CronTool initialization failed: %w", err)
		}

		agentLoop.RegisterTool(cronTool)
	}

	// Set onJob handler
	if cronTool != nil {
		cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
			result := cronTool.ExecuteJob(context.Background(), job)
			return result, nil
		})
	}

	return cronService, nil
}
