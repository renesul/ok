package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

const SessionCookieName = "ok_session"

func RequireAuth(sessionService *application.SessionService, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Cookies(SessionCookieName)
		if sessionID == "" {
			return handleUnauthenticated(c)
		}

		valid, err := sessionService.ValidateSession(c.Context(), sessionID)
		if err != nil {
			log.Debug("session validation error", zap.Error(err))
			return handleUnauthenticated(c)
		}
		if !valid {
			return handleUnauthenticated(c)
		}

		return c.Next()
	}
}

func handleUnauthenticated(c *fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	return c.Redirect("/login")
}
