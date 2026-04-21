package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
)

type mongoContentRepository interface {
	List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error)
	GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
}

type ContentsHandler struct {
	contentRepo mongoContentRepository
}

func NewContentsHandler(contentRepo mongoContentRepository) *ContentsHandler {
	return &ContentsHandler{contentRepo: contentRepo}
}

func (h *ContentsHandler) List(c echo.Context) error {
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.ParseInt(c.QueryParam("offset"), 10, 64)

	var contentType *domain.ContentType
	if ct := c.QueryParam("content_type"); ct != "" {
		v := domain.ContentType(ct)
		contentType = &v
	}

	params := mongoRepo.ListParams{
		ContentType: contentType,
		Limit:       limit,
		Offset:      offset,
	}

	contents, err := h.contentRepo.List(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if contents == nil {
		contents = []*domain.MongoContent{}
	}
	return c.JSON(http.StatusOK, contents)
}

func (h *ContentsHandler) Get(c echo.Context) error {
	shortID := c.Param("short_id")
	content, err := h.contentRepo.GetByShortID(c.Request().Context(), shortID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return echo.NewHTTPError(http.StatusNotFound, "content not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, content)
}

func (h *ContentsHandler) GetVariants(c echo.Context) error {
	shortID := c.Param("short_id")
	variants, err := h.contentRepo.GetVariantsByShortID(c.Request().Context(), shortID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return echo.NewHTTPError(http.StatusNotFound, "content not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if variants == nil {
		variants = []domain.MongoVariant{}
	}
	return c.JSON(http.StatusOK, variants)
}
