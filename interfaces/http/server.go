package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/interfaces/http/handler"
	"github.com/renesul/ok/interfaces/http/middleware"
	"go.uber.org/zap"
)

func NewServer(userHandler *handler.UserHandler, log *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(middleware.Recovery(log))
	app.Use(middleware.Logger(log))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api")
	users := api.Group("/users")
	users.Post("/", userHandler.Create)
	users.Get("/", userHandler.List)
	users.Get("/:id", userHandler.Get)
	users.Put("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)

	return app
}
