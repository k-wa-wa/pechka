package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"pechka/streaming-service/api/internal/infrastructure/auth"
)

// RequireAppJWT extracts the Bearer token, verifies it, and stores claims in c.Locals("user")
func RequireAppJWT() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed Authorization header"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.VerifyAppJWT(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid access token"})
		}

		c.Locals("user", claims)
		return c.Next()
	}
}

// RequirePermission checks if the authenticated user has the required permission mapping (resource:action) or 'admin' role.
// Must be used after RequireAppJWT.
func RequirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("user").(*auth.AppClaims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		// Allow admins bypass
		for _, role := range claims.Roles {
			if role == "admin" || role == "Administrator" {
				return c.Next()
			}
		}

		// Check specific permission
		requiredPerm := resource + ":" + action
		for _, p := range claims.Permissions {
			if p == requiredPerm {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden: insufficient permissions"})
	}
}
