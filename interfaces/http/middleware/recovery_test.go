package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func TestRecoveryMiddleware_PanicReturns500(t *testing.T) {
	app := fiber.New()
	app.Use(Recovery(zap.NewNop()))
	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500 after panic, got %d", resp.StatusCode)
	}
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	app := fiber.New()
	app.Use(Recovery(zap.NewNop()))
	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
