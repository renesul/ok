package integration

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	agent "github.com/renesul/ok/infrastructure/agent"
	sched "github.com/renesul/ok/infrastructure/scheduler"
	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/embedding"
	"github.com/renesul/ok/infrastructure/llm"
	"github.com/renesul/ok/infrastructure/repository"
	apphttp "github.com/renesul/ok/interfaces/http"
	"github.com/renesul/ok/interfaces/http/handler"
	"github.com/renesul/ok/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	testDB          *gorm.DB
	testApp         *fiber.App
	testCfg         *config.Config
	testAgentMemory *agent.SQLiteMemory
	testExecRepo    *agent.ExecutionRepository
	testConfigRepo  *agent.ConfigRepository
)

func TestMain(m *testing.M) {
	var err error
	testCfg, err = config.LoadFrom("../../data/.env.test")
	if err != nil {
		panic("load test config: " + err.Error())
	}

	testDB, err = database.New(testCfg.DatabasePath, testCfg.Debug)
	if err != nil {
		panic("open test database: " + err.Error())
	}

	if err := database.RunMigrations(testDB); err != nil {
		panic("run test migrations: " + err.Error())
	}

	log := zap.NewNop()

	// Repositories
	sessionRepository := repository.NewSessionRepository(testDB, log)
	conversationRepository := repository.NewConversationRepository(testDB, log)
	messageRepository := repository.NewMessageRepository(testDB, log)

	// Infrastructure
	llmClient := llm.NewClient(log)
	embedClient := embedding.NewClient(embedding.ClientConfig{
		Provider: testCfg.EmbedProvider,
		BaseURL:  testCfg.EmbedBaseURL,
		APIKey:   testCfg.EmbedAPIKey,
		Model:    testCfg.EmbedModel,
	}, log)

	// Services
	sessionService := application.NewSessionService(sessionRepository, log)
	embeddingService := application.NewEmbeddingService(embedClient, messageRepository, conversationRepository, log)
	conversationService := application.NewConversationService(conversationRepository, messageRepository, embeddingService, log)
	importService := application.NewImportService(conversationRepository, messageRepository, embeddingService, log)
	// Agent (antes do chat)
	planner := agent.NewDefaultPlanner(log)
	planner.RegisterTool(&agenttools.EchoTool{})
	planner.RegisterTool(agenttools.NewHTTPTool())
	agentExecutor := agent.NewDefaultExecutor(log)
	agentMemory := agent.NewSQLiteMemory(testDB, log)
	testAgentMemory = agentMemory
	execRepo := agent.NewExecutionRepository(testDB, log)
	testExecRepo = execRepo
	agentConfigRepo := agent.NewConfigRepository(testDB, log)
	testConfigRepo = agentConfigRepo
	llmConfig := llm.ClientConfig{
		BaseURL: testCfg.LLMBaseURL,
		APIKey:  testCfg.LLMAPIKey,
		Model:   testCfg.LLMModel,
	}
	llmFastConfig := llm.ClientConfig{
		BaseURL: testCfg.LLMFastBaseURL,
		APIKey:  testCfg.LLMFastAPIKey,
		Model:   testCfg.LLMFastModel,
	}

	agentService := application.NewAgentService(testDB, llmClient, llmConfig, llmFastConfig, planner, agentExecutor, agentMemory, execRepo, agentConfigRepo, log)

	llmConfigured := testCfg.LLMBaseURL != "" && testCfg.LLMAPIKey != "" && testCfg.LLMModel != ""
	chatService := application.NewChatService(conversationRepository, messageRepository, embeddingService, agentService, llmConfigured, log)

	// Handlers
	authHandler := handler.NewAuthHandler(sessionService, testCfg.AuthPassword, log)
	chatHandler := handler.NewChatHandler(conversationService, chatService, log)
	importHandler := handler.NewImportHandler(importService, log)
	healthHandler := handler.NewHealthHandler(llmClient, llm.ClientConfig{
		BaseURL: testCfg.LLMBaseURL,
		APIKey:  testCfg.LLMAPIKey,
		Model:   testCfg.LLMModel,
	}, embedClient, log)
	agentHandler := handler.NewAgentHandler(agentService, nil, handler.ChannelStatus{}, log)

	// Scheduler
	jobRepository := sched.NewJobRepository(testDB, log)
	schedulerService := application.NewSchedulerService(jobRepository, log)
	schedulerHandler := handler.NewSchedulerHandler(schedulerService, log)

	wsHandler := handler.NewWSHandler(agentService, nil, log)
	testApp = apphttp.NewServer(authHandler, chatHandler, importHandler, healthHandler, agentHandler, schedulerHandler, wsHandler, sessionService, testCfg, log)

	code := m.Run()

	os.Remove(testCfg.DatabasePath)
	os.Exit(code)
}

func cleanupSessions(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM sessions")
}

func cleanupJobs(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM scheduled_jobs")
}

func cleanupFeedback(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM agent_feedback")
}

func cleanupConversations(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM message_embeddings")
	testDB.Exec("DELETE FROM messages_fts")
	testDB.Exec("DELETE FROM messages")
	testDB.Exec("DELETE FROM conversations")
}

func cleanupMemory(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM agent_memory")
}

func cleanupExecutions(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM agent_executions")
}

func cleanupAudit(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM agent_audit")
}

func cleanupAll(t *testing.T) {
	t.Helper()
	cleanupConversations(t)
	cleanupSessions(t)
}

func loginAndGetCookie(t *testing.T) string {
	t.Helper()
	body := `{"password":"` + testCfg.AuthPassword + `"}`
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(respBody))
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "ok_session" {
			return cookie.Value
		}
	}
	t.Fatal("no session cookie found")
	return ""
}

func authenticatedRequest(t *testing.T, method, path string, body io.Reader) *http.Response {
	t.Helper()
	cookie := loginAndGetCookie(t)
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Cookie", "ok_session="+cookie)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func unauthenticatedRequest(t *testing.T, method, path string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}
