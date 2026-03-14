package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/usecase"
)

type ContentHandler struct {
	useCase usecase.ContentUseCase
}

func NewContentHandler(useCase usecase.ContentUseCase) *ContentHandler {
	return &ContentHandler{useCase: useCase}
}

func (h *ContentHandler) RegisterRoutes(router fiber.Router) {
	adminGroup := router.Group("/admin/metadata")
	
	// Video routes
	adminGroup.Get("/videos", h.ListVideos)
	adminGroup.Post("/videos", h.CreateVideo)
	adminGroup.Get("/videos/:short_id", h.GetVideo)
	adminGroup.Put("/videos/:id", h.UpdateVideo)

	// Gallery routes
	adminGroup.Get("/galleries", h.ListGalleries)
	adminGroup.Post("/galleries", h.CreateGallery)
	adminGroup.Get("/galleries/:short_id", h.GetGallery)
	adminGroup.Put("/galleries/:id", h.UpdateGallery)

	// Common
	adminGroup.Post("/assets/:id", h.AddAssets)
}

func (h *ContentHandler) CreateVideo(c *fiber.Ctx) error {
	var req usecase.CreateVideoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	video, err := h.useCase.CreateVideo(c.Context(), req)
	if err != nil {
		log.Printf("Failed to create video: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create video"})
	}

	return c.Status(fiber.StatusCreated).JSON(video)
}

func (h *ContentHandler) CreateGallery(c *fiber.Ctx) error {
	var req usecase.CreateGalleryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	gallery, err := h.useCase.CreateGallery(c.Context(), req)
	if err != nil {
		log.Printf("Failed to create gallery: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create gallery"})
	}

	return c.Status(fiber.StatusCreated).JSON(gallery)
}

func (h *ContentHandler) AddAssets(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
	}

	var req usecase.AddAssetsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if err := h.useCase.AddAssets(c.Context(), id, req); err != nil {
		log.Printf("Failed to add assets: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add assets"})
	}

	return c.JSON(fiber.Map{"message": "assets added successfully"})
}

func (h *ContentHandler) GetVideo(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	video, err := h.useCase.GetVideoDetails(c.Context(), shortID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Video not found"})
	}
	return c.JSON(video)
}

func (h *ContentHandler) GetGallery(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	gallery, err := h.useCase.GetGalleryDetails(c.Context(), shortID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Gallery not found"})
	}
	return c.JSON(gallery)
}

func (h *ContentHandler) ListVideos(c *fiber.Ctx) error {
	videos, err := h.useCase.ListVideos(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list videos"})
	}
	if videos == nil {
		videos = []*domain.Video{}
	}
	return c.JSON(videos)
}

func (h *ContentHandler) ListGalleries(c *fiber.Ctx) error {
	galleries, err := h.useCase.ListGalleries(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list galleries"})
	}
	if galleries == nil {
		galleries = []*domain.Gallery{}
	}
	return c.JSON(galleries)
}

func (h *ContentHandler) UpdateVideo(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var v domain.Video
	if err := c.BodyParser(&v); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.useCase.UpdateVideo(c.Context(), id, &v); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update video"})
	}
	return c.JSON(v)
}

func (h *ContentHandler) UpdateGallery(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var g domain.Gallery
	if err := c.BodyParser(&g); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.useCase.UpdateGallery(c.Context(), id, &g); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update gallery"})
	}
	return c.JSON(g)
}
