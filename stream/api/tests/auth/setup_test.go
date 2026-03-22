package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	api "pechka/streaming-service/api/internal/interface/api"
	"pechka/streaming-service/api/internal/usecase"
)

// mockAuthUseCase is a configurable test double for usecase.AuthUseCase.
type mockAuthUseCase struct {
	sessionFn func(ctx context.Context, token string) (*usecase.SessionResponse, error)
	meFn      func(ctx context.Context, token string) (*usecase.MeResponse, error)
}

func (m *mockAuthUseCase) Session(ctx context.Context, token string) (*usecase.SessionResponse, error) {
	if m.sessionFn != nil {
		return m.sessionFn(ctx, token)
	}
	return nil, nil
}

func (m *mockAuthUseCase) Me(ctx context.Context, token string) (*usecase.MeResponse, error) {
	if m.meFn != nil {
		return m.meFn(ctx, token)
	}
	return nil, nil
}

// newApp builds a Fiber app with auth routes registered.
func newApp(uc usecase.AuthUseCase) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	api.NewAuthHandler(uc).RegisterRoutes(app.Group("/api/v1"))
	return app
}

func assertStatus(t *testing.T, expected int, resp *http.Response) {
	t.Helper()
	if expected != resp.StatusCode {
		t.Errorf("status: expected %d, got %d", expected, resp.StatusCode)
	}
}

func assertLocation(t *testing.T, expected string, resp *http.Response) {
	t.Helper()
	if loc := resp.Header.Get("Location"); loc != expected {
		t.Errorf("Location: expected %q, got %q", expected, loc)
	}
}
