package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/usecase"
)

type MetadataHandler struct {
	contentUC usecase.ContentUseCase
	userRepo  domain.UserRepository
}

func NewMetadataHandler(contentUC usecase.ContentUseCase, userRepo domain.UserRepository) *MetadataHandler {
	return &MetadataHandler{
		contentUC: contentUC,
		userRepo:  userRepo,
	}
}

func (h *MetadataHandler) RegisterRoutes(router fiber.Router) {
	admin := router.Group("/admin/metadata")
	admin.Get("/contents", h.ListContents)
	admin.Get("/contents/:id", h.GetContent)
	admin.Put("/contents/:id", h.UpdateContent)
	admin.Get("/groups", h.ListGroups)
}

// GET /admin/metadata/contents
func (h *MetadataHandler) ListContents(c *fiber.Ctx) error {
	contents, err := h.contentUC.ListContents(c.Context())
	if err != nil {
		log.Printf("ListContents error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list contents"})
	}
	if contents == nil {
		contents = []*domain.Content{}
	}
	return c.JSON(contents)
}

// GET /admin/metadata/contents/:id
func (h *MetadataHandler) GetContent(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
	}

	content, err := h.contentUC.GetContentByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}
	return c.JSON(content)
}

// PUT /admin/metadata/contents/:id
func (h *MetadataHandler) UpdateContent(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
	}

	var body usecase.CreateContentRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updated, err := h.contentUC.UpdateContent(c.Context(), id, body)
	if err != nil {
		log.Printf("UpdateContent error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(updated)
}

// GET /admin/metadata/groups
func (h *MetadataHandler) ListGroups(c *fiber.Ctx) error {
	groups, err := h.userRepo.ListAllGroups(c.Context())
	if err != nil {
		log.Printf("ListGroups error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list groups"})
	}
	if groups == nil {
		groups = []domain.Group{}
	}
	return c.JSON(groups)
}
