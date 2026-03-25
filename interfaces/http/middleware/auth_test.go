package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type mockSessionRepo struct {
	session *domain.Session
}

func (r *mockSessionRepo) Create(_ context.Context, _ *domain.Session) error { return nil }
func (r *mockSessionRepo) FindByID(_ context.Context, _ string) (*domain.Session, error) {
	return r.session, nil
}
func (r *mockSessionRepo) DeleteByID(_ context.Context, _ string) error { return nil }
func (r *mockSessionRepo) DeleteExpired(_ context.Context) error        { return nil }

func setupTestApp(sessionRepo domain.SessionRepository) *fiber.App {
	svc := application.NewSessionService(sessionRepo, zap.NewNop())
	app := fiber.New()
	app.Use(RequireAuth(svc, zap.NewNop()))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/page", func(c *fiber.Ctx) error {
		return c.SendString("page")
	})
	return app
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	repo := &mockSessionRepo{
		session: &domain.Session{
			ID:        "valid-session",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}
	app := setupTestApp(repo)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Cookie", SessionCookieName+"=valid-session")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for valid session, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_NoSession_API(t *testing.T) {
	app := setupTestApp(&mockSessionRepo{})

	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for missing session on API, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_NoSession_Page(t *testing.T) {
	app := setupTestApp(&mockSessionRepo{})

	req := httptest.NewRequest("GET", "/page", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302 redirect for missing session on page, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	repo := &mockSessionRepo{
		session: &domain.Session{
			ID:        "expired",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		},
	}
	app := setupTestApp(repo)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Cookie", SessionCookieName+"=expired")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for expired session, got %d", resp.StatusCode)
	}
}
