package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/interfaces/http/middleware"
	"go.uber.org/zap"
)

type AuthHandler struct {
	sessionService *application.SessionService
	authPassword   string
	log            *zap.Logger
}

func NewAuthHandler(sessionService *application.SessionService, authPassword string, log *zap.Logger) *AuthHandler {
	return &AuthHandler{
		sessionService: sessionService,
		authPassword:   authPassword,
		log:            log.Named("handler.auth"),
	}
}

func (h *AuthHandler) LoginPage(c *fiber.Ctx) error {
	sessionID := c.Cookies(middleware.SessionCookieName)
	if sessionID != "" {
		valid, _ := h.sessionService.ValidateSession(c.Context(), sessionID)
		if valid {
			return c.Redirect("/agent")

		}
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(readTemplate("login.html"))
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.log.Debug("invalid login request", zap.Error(err))
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	h.log.Debug("login attempt", zap.Int("password_length", len(req.Password)), zap.Int("expected_length", len(h.authPassword)))

	if req.Password != h.authPassword {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "senha invalida",
		})
	}

	session, err := h.sessionService.CreateSession(c.Context())
	if err != nil {
		h.log.Debug("create session failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create session",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"message": "login successful",
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := c.Cookies(middleware.SessionCookieName)
	if sessionID != "" {
		if err := h.sessionService.DestroySession(c.Context(), sessionID); err != nil {
			h.log.Debug("destroy session failed", zap.Error(err))
		}
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"message": "logged out",
	})
}

func (h *AuthHandler) AgentPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(readTemplate("agent.html"))
}

func (h *AuthHandler) ProfilePage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(readTemplate("profile.html"))
}
