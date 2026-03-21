package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// MockAuthMiddleware is used for local development to simulate Cloudflare Access authentication.
// When MOCK_AUTH_ENABLED=true, it reads X-Mock-Email header or mock_email query param
// and injects it into Cf-Access-Authenticated-User-Email.
func MockAuthMiddleware() fiber.Handler {
	isMockEnabled := os.Getenv("MOCK_AUTH_ENABLED") == "true"

	return func(c *fiber.Ctx) error {
		if !isMockEnabled {
			return c.Next()
		}

		mockEmail := c.Get("X-Mock-Email")
		if mockEmail == "" {
			mockEmail = c.Query("mock_email")
		}

		if mockEmail != "" {
			c.Request().Header.Set("Cf-Access-Authenticated-User-Email", mockEmail)
		}

		return c.Next()
	}
}
