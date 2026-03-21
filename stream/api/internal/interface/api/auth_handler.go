package api

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"pechka/streaming-service/api/internal/usecase"
)

type AuthHandler struct {
	uc usecase.AuthUseCase
}

func NewAuthHandler(uc usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

func (h *AuthHandler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Get("/session", h.Session)
	auth.Get("/me", h.Me)
}

// GET /api/v1/auth/session
func (h *AuthHandler) Session(c *fiber.Ctx) error {
	tokenStr := c.Get("Cf-Access-Jwt-Assertion")
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing Cloudflare authentication token"})
	}

	res, err := h.uc.Session(c.Context(), tokenStr)
	if err != nil {
		log.Printf("session error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to establish session"})
	}

	return c.JSON(res)
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed Authorization header"})
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	me, err := h.uc.Me(c.Context(), accessToken)
	if err != nil {
		log.Printf("me error: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid access token"})
	}

	return c.JSON(me)
}
