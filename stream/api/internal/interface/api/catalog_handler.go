package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"pechka/streaming-service/api/internal/domain"
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
	catalog := router.Group("/catalog")
	catalog.Get("/home", h.GetHome)
	catalog.Get("/contents/:short_id", h.GetDetails)

	// Internal Sync API
	internal := router.Group("/internal/catalog")
	internal.Post("/sync/:short_id", h.Sync)
}

func (h *CatalogHandler) GetHome(c *fiber.Ctx) error {
	res, err := h.uc.GetHome(c.Context())
	if err != nil {
		log.Printf("GetHome error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (h *CatalogHandler) GetDetails(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	res, err := h.uc.GetContentDetails(c.Context(), shortID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
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
