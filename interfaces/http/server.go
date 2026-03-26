package http

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/websocket/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/internal/config"
	"github.com/renesul/ok/interfaces/http/handler"
	"github.com/renesul/ok/interfaces/http/middleware"
	"github.com/renesul/ok/web"
	"go.uber.org/zap"
)

func NewServer(
	authHandler *handler.AuthHandler,
	chatHandler *handler.ChatHandler,
	importHandler *handler.ImportHandler,
	healthHandler *handler.HealthHandler,
	agentHandler *handler.AgentHandler,
	schedulerHandler *handler.SchedulerHandler,
	wsHandler *handler.WSHandler,
	sessionService *application.SessionService,
	cfg *config.Config,
	log *zap.Logger,
) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             100 * 1024 * 1024,
	})

	app.Use(middleware.Recovery(log))
	app.Use(middleware.Logger(log))

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(web.Static),
		PathPrefix: "static",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Public routes
	app.Get("/login", authHandler.LoginPage)
	app.Post("/api/auth/login", authHandler.Login)

	// Protected routes
	auth := middleware.RequireAuth(sessionService, log)

	app.Get("/", auth, func(c *fiber.Ctx) error {
		return c.Redirect("/agent")

	})
	app.Get("/chat", auth, chatHandler.ChatPage)
	app.Get("/chat/:id", auth, chatHandler.ChatPage)
	app.Get("/agent", auth, authHandler.AgentPage)
	app.Get("/profile", auth, authHandler.ProfilePage)
	app.Get("/profile/*", auth, authHandler.ProfilePage)

	api := app.Group("/api", auth)

	api.Post("/auth/logout", authHandler.Logout)
	api.Get("/health/services", healthHandler.CheckServices)

	conversations := api.Group("/conversations")
	conversations.Get("/", chatHandler.ListConversations)
	conversations.Get("/search", chatHandler.SearchConversations)
	conversations.Post("/", chatHandler.CreateConversation)
	conversations.Get("/:id/messages", chatHandler.GetMessages)
	conversations.Post("/:id/messages", chatHandler.SendMessage)
	conversations.Delete("/:id", chatHandler.DeleteConversation)

	api.Post("/import/chatgpt", importHandler.ImportChatGPT)
	api.Post("/agent/run", agentHandler.Run)
	api.Post("/agent/stream", agentHandler.Stream)
	api.Get("/agent/status", agentHandler.Status)
	api.Get("/agent/executions", agentHandler.ListExecutions)
	api.Get("/agent/executions/:id", agentHandler.GetExecution)
	api.Get("/agent/tools", agentHandler.ListTools)
	api.Get("/agent/skills", agentHandler.ListSkills)
	api.Get("/agent/metrics", agentHandler.Metrics)
	api.Get("/agent/limits", agentHandler.GetLimits)
	api.Put("/agent/limits", agentHandler.SetLimits)
	api.Get("/agent/config/:key", agentHandler.GetConfig)
	api.Post("/agent/confirm/:id", agentHandler.Confirm)
	api.Post("/agent/cancel", agentHandler.Cancel)

	// WebSocket (auth via cookie checked in upgrade middleware)
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/agent", auth, websocket.New(wsHandler.Handle))

	api.Get("/config", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"server_port":    cfg.ServerPort,
			"debug":          cfg.Debug,
			"llm_base_url":      cfg.LLMBaseURL,
			"llm_model":         cfg.LLMModel,
			"llm_fast_base_url": cfg.LLMFastBaseURL,
			"llm_fast_model":    cfg.LLMFastModel,
			"vision_base_url":   cfg.VisionBaseURL,
			"vision_model":      cfg.VisionModel,
			"embed_provider":    cfg.EmbedProvider,
			"embed_base_url": cfg.EmbedBaseURL,
			"embed_model":    cfg.EmbedModel,
			"agent_sandbox":  cfg.AgentSandboxDir,
		})
	})
	api.Put("/agent/config/:key", agentHandler.SetConfig)

	schedulerRoutes := api.Group("/scheduler")
	schedulerRoutes.Get("/jobs", schedulerHandler.List)
	schedulerRoutes.Post("/jobs", schedulerHandler.Create)
	schedulerRoutes.Put("/jobs/:id", schedulerHandler.Update)
	schedulerRoutes.Delete("/jobs/:id", schedulerHandler.Delete)

	return app
}
