package api

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/infrastructure/auth"
	"pechka/streaming-service/api/internal/usecase"
)

type CatalogHandler struct {
	uc       usecase.CatalogUseCase
	metaRepo domain.ContentRepository // To fetch from PG during sync
}

func NewCatalogHandler(uc usecase.CatalogUseCase, metaRepo domain.ContentRepository) *CatalogHandler {
	return &CatalogHandler{
		uc:       uc,
		metaRepo: metaRepo,
	}
}

func (h *CatalogHandler) RegisterRoutes(router fiber.Router) {
	// The auth middleware (RequireAppJWT) should be applied before these routes in `run.go`
	// or locally here. For now, assuming it's applied to the `/catalog` group.
	catalog := router.Group("/catalog")
	catalog.Get("/home", h.GetHome)
	catalog.Get("/contents/:short_id", h.GetDetails)
	catalog.Get("/search", h.Search)

	// Internal Sync API
	internal := router.Group("/internal/catalog")
	internal.Post("/sync/:short_id", h.Sync)
}

func getUserGroups(c *fiber.Ctx) []string {
	if claims, ok := c.Locals("user").(*auth.AppClaims); ok && claims != nil && claims.Groups != nil {
		return claims.Groups
	}
	return []string{} // Anonymous viewing (only public contents)
}

func (h *CatalogHandler) GetHome(c *fiber.Ctx) error {
	groups := getUserGroups(c)
	res, err := h.uc.GetHome(c.Context(), groups)
	if err != nil {
		log.Printf("GetHome error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (h *CatalogHandler) GetDetails(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	groups := getUserGroups(c)
	res, err := h.uc.GetContentDetails(c.Context(), shortID, groups)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found or access denied"})
	}
	return c.JSON(res)
}

func (h *CatalogHandler) Sync(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	if err := h.uc.SyncContent(c.Context(), shortID, h.metaRepo); err != nil {
		log.Printf("sync error for %s: %v", shortID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "sync failed"})
	}
	return c.JSON(fiber.Map{"status": "synchronized"})
}

func (h *CatalogHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	tagsStr := c.Query("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	groups := getUserGroups(c)

	res, err := h.uc.Search(c.Context(), query, tags, groups)
	if err != nil {
		log.Printf("Search error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "search failed"})
	}
	return c.JSON(res)
}
