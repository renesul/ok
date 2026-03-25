package bootstrap

import (
	"context"
	"database/sql"

	"github.com/renesul/ok/application"
	"github.com/renesul/ok/application/engine"
	agent "github.com/renesul/ok/infrastructure/agent"
	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	"github.com/renesul/ok/infrastructure/embedding"
	"github.com/renesul/ok/infrastructure/llm"
	"github.com/renesul/ok/infrastructure/security"
	sched "github.com/renesul/ok/infrastructure/scheduler"
	"github.com/renesul/ok/internal/config"
	"go.uber.org/zap"
)

type Components struct {
	DB                 *sql.DB
	LLMClient          *llm.Client
	EmbedClient        *embedding.Client
	AgentService       *application.AgentService
	AgentMemory        *agent.SQLiteMemory
	JobRepository      *sched.JobRepository
	ConfirmManager     *agent.ConfirmationManager
	LLMHeavy          llm.ClientConfig
	LLMFast            llm.ClientConfig
}

func NewAgent(db *sql.DB, cfg *config.Config, log *zap.Logger) *Components {
	scrubber := security.NewSecretScrubber()

	llmClient := llm.NewClient(log)
	llmClient.SetScrubber(scrubber)
	embedClient := embedding.NewClient(embedding.ClientConfig{
		Provider: cfg.EmbedProvider,
		BaseURL:  cfg.EmbedBaseURL,
		APIKey:   cfg.EmbedAPIKey,
		Model:    cfg.EmbedModel,
	}, log)

	jobRepository := sched.NewJobRepository(db, log)

	planner := agent.NewDefaultPlanner(log)
	confirmManager := agent.NewConfirmationManager()

	planner.RegisterTool(&agenttools.EchoTool{})
	planner.RegisterTool(agenttools.NewHTTPTool())
	planner.RegisterTool(agenttools.NewFileReadTool(cfg.AgentSandboxDir))
	planner.RegisterTool(agenttools.NewFileWriteTool(cfg.AgentSandboxDir))
	planner.RegisterTool(agenttools.NewShellToolWithConfirmation(confirmManager))
	planner.RegisterTool(&agenttools.JSONParseTool{})
	planner.RegisterTool(&agenttools.Base64Tool{})
	planner.RegisterTool(&agenttools.TimestampTool{})
	planner.RegisterTool(&agenttools.MathTool{})
	planner.RegisterTool(&agenttools.TextExtractTool{})
	planner.RegisterTool(agenttools.NewIndexFolderTool(cfg.AgentSandboxDir))
	planner.RegisterTool(agenttools.NewScheduleTaskTool(jobRepository))
	planner.RegisterTool(agenttools.NewSearchTool(cfg.AgentSandboxDir))
	planner.RegisterTool(agenttools.NewFileEditTool(confirmManager))
	visionConfig := llm.ClientConfig{
		BaseURL: cfg.VisionBaseURL,
		APIKey:  cfg.VisionAPIKey,
		Model:   cfg.VisionModel,
	}
	planner.RegisterTool(agenttools.NewBrowserTool(llmClient, visionConfig))
	planner.RegisterTool(agenttools.NewREPLTool(confirmManager))
	planner.RegisterTool(agenttools.NewWebSearchTool())

	agentMemory := agent.NewSQLiteMemory(db, log)
	agentMemory.SetEmbeddingClient(embedClient)
	agentMemory.SetScrubber(scrubber)
	planner.RegisterTool(agenttools.NewLearnRuleTool(agentMemory))

	executor := agent.NewDefaultExecutor(log)
	auditLog := agent.NewAuditLog(db, log)
	executor.SetAuditLog(auditLog)

	execRepo := agent.NewExecutionRepository(db, log)
	agentConfigRepo := agent.NewConfigRepository(db, log)

	planner.RegisterTool(agenttools.NewSqlInspectorTool(agentConfigRepo))
	planner.RegisterTool(agenttools.NewPythonRPATool(cfg.AgentSandboxDir, confirmManager))
	planner.RegisterTool(agenttools.NewGmailReadTool(agentConfigRepo))
	planner.RegisterTool(agenttools.NewGmailSendTool(agentConfigRepo))
	planner.RegisterTool(agenttools.NewGCalManagerTool(agentConfigRepo))
	planner.RegisterTool(agenttools.NewDockerReplicatorTool(confirmManager))
	planner.RegisterTool(agenttools.NewConfigTool(agentConfigRepo))
	planner.RegisterTool(agenttools.NewSkillCreatorTool(cfg.AgentSandboxDir))
	skillRepo := agent.NewFileSkillRepository(cfg.AgentSandboxDir)
	planner.RegisterTool(agenttools.NewSkillLoaderTool(skillRepo))

	llmHeavy := llm.ClientConfig{
		BaseURL:          cfg.LLMBaseURL,
		APIKey:           cfg.LLMAPIKey,
		Model:            cfg.LLMModel,
		MaxContextTokens: 128000,
	}
	llmFast := llm.ClientConfig{
		BaseURL:          cfg.LLMFastBaseURL,
		APIKey:           cfg.LLMFastAPIKey,
		Model:            cfg.LLMFastModel,
		MaxContextTokens: 8192,
	}

	// Sub-engine factory para DelegateTaskTool (sem delegate — previne recursao)
	subPlanner := agent.NewDefaultPlanner(log)
	for name, tool := range planner.Tools() {
		if name != "delegate" {
			subPlanner.RegisterTool(tool)
		}
	}

	agentService := application.NewAgentService(
		db, llmClient, llmHeavy, llmFast,
		planner, executor, agentMemory,
		execRepo, agentConfigRepo, skillRepo,
		log,
	)

	subEngineRunner := func(ctx context.Context, input string) ([]string, error) {
		subEngine := engine.NewAgentEngine(
			db, llmClient, llmHeavy, llmFast,
			subPlanner, executor, agentMemory, execRepo,
			agentService.GetLimits(), agentService.BuildSystemPrompt, log,
		)
		emitter := engine.NewBufferEmitter()
		if err := subEngine.RunLoop(ctx, input, emitter); err != nil {
			return nil, err
		}
		return emitter.Response().Messages, nil
	}
	planner.RegisterTool(agenttools.NewDelegateTaskTool(subEngineRunner))

	return &Components{
		DB:                 db,
		LLMClient:          llmClient,
		EmbedClient:        embedClient,
		AgentService:       agentService,
		AgentMemory:        agentMemory,
		JobRepository:      jobRepository,
		ConfirmManager:     confirmManager,
		LLMHeavy:          llmHeavy,
		LLMFast:            llmFast,
	}
}

func (c *Components) LLMConfigured() bool {
	return c.LLMHeavy.BaseURL != "" && c.LLMHeavy.APIKey != "" && c.LLMHeavy.Model != ""
}
