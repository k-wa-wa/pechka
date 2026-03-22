package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/usecase"
)

func (h *ContentHandler) ListGroups(c *fiber.Ctx) error {
	groups, err := h.groupLister.ListAllGroups(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list groups"})
	}
	if groups == nil {
		groups = []domain.Group{}
	}
	return c.JSON(groups)
}

// ContentHandler handles admin metadata API for content management.
type ContentHandler struct {
	useCase     usecase.ContentUseCase
	groupLister domain.UserRepository
}

func NewContentHandler(useCase usecase.ContentUseCase, groupLister domain.UserRepository) *ContentHandler {
	return &ContentHandler{useCase: useCase, groupLister: groupLister}
}

func (h *ContentHandler) RegisterRoutes(router fiber.Router) {
	admin := router.Group("/admin/metadata")

	admin.Get("/contents", h.ListContents)
	admin.Post("/contents", h.CreateContent)
	admin.Get("/contents/:short_id", h.GetContent)
	admin.Put("/contents/:id", h.UpdateContent)
	admin.Post("/contents/:id/assets", h.AddAssets)
	admin.Get("/groups", h.ListGroups)
}

func (h *ContentHandler) CreateContent(c *fiber.Ctx) error {
	var req usecase.CreateContentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	if req.ContentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content_type is required"})
	}

	content, err := h.useCase.CreateContent(c.Context(), req)
	if err != nil {
		log.Printf("failed to create content: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create content"})
	}
	return c.Status(fiber.StatusCreated).JSON(content)
}

func (h *ContentHandler) GetContent(c *fiber.Ctx) error {
	shortID := c.Params("short_id")
	content, err := h.useCase.GetContentDetails(c.Context(), shortID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}
	return c.JSON(content)
}

func (h *ContentHandler) ListContents(c *fiber.Ctx) error {
	contents, err := h.useCase.ListContents(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list contents"})
	}
	if contents == nil {
		contents = []*domain.Content{}
	}
	return c.JSON(contents)
}

func (h *ContentHandler) UpdateContent(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req usecase.CreateContentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	content, err := h.useCase.UpdateContent(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update content"})
	}
	return c.JSON(content)
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
		log.Printf("failed to add assets: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add assets"})
	}
	return c.JSON(fiber.Map{"message": "assets added successfully"})
}
