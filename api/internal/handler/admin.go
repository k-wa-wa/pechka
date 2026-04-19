package handler

import (
	"net/http"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type AdminHandler struct {
	contentRepo *pgRepo.ContentRepository
	discRepo    *pgRepo.DiscRepository
	snowflake   *snowflake.Node
}

func NewAdminHandler(contentRepo *pgRepo.ContentRepository, discRepo *pgRepo.DiscRepository, node *snowflake.Node) *AdminHandler {
	return &AdminHandler{contentRepo: contentRepo, discRepo: discRepo, snowflake: node}
}

type createContentRequest struct {
	ContentType     domain.ContentType `json:"content_type" validate:"required"`
	DiscID          *string            `json:"disc_id"`
	Title           string             `json:"title" validate:"required"`
	Description     string             `json:"description"`
	DurationSeconds *int               `json:"duration_seconds"`
	Is360           bool               `json:"is_360"`
	Tags            []string           `json:"tags"`
}

func (h *AdminHandler) CreateContent(c echo.Context) error {
	var req createContentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Title == "" || req.ContentType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and content_type are required")
	}

	shortID := h.snowflake.Generate().String()
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	content, err := h.contentRepo.Create(c.Request().Context(), pgRepo.CreateContentParams{
		ShortID:         shortID,
		ContentType:     req.ContentType,
		DiscID:          req.DiscID,
		Title:           req.Title,
		Description:     req.Description,
		DurationSeconds: req.DurationSeconds,
		Is360:           req.Is360,
		Tags:            tags,
		Status:          domain.ContentStatusPending,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, content)
}

type updateContentRequest struct {
	Title       *string               `json:"title"`
	Description *string               `json:"description"`
	Tags        []string              `json:"tags"`
	Status      *domain.ContentStatus `json:"status"`
}

func (h *AdminHandler) UpdateContent(c echo.Context) error {
	id := c.Param("id")
	var req updateContentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	content, err := h.contentRepo.Update(c.Request().Context(), pgRepo.UpdateContentParams{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		Status:      req.Status,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "content not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, content)
}

func (h *AdminHandler) DeleteContent(c echo.Context) error {
	id := c.Param("id")
	if err := h.contentRepo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) ListContents(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	var status *domain.ContentStatus
	if s := c.QueryParam("status"); s != "" {
		v := domain.ContentStatus(s)
		status = &v
	}

	contents, err := h.contentRepo.List(c.Request().Context(), pgRepo.ListContentsParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if contents == nil {
		contents = []*domain.Content{}
	}
	return c.JSON(http.StatusOK, contents)
}

func (h *AdminHandler) ListDiscs(c echo.Context) error {
	discs, err := h.discRepo.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if discs == nil {
		discs = []*domain.Disc{}
	}
	return c.JSON(http.StatusOK, discs)
}

type createDiscRequest struct {
	Label    string  `json:"label"`
	DiscName *string `json:"disc_name"`
}

func (h *AdminHandler) CreateDisc(c echo.Context) error {
	var req createDiscRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Label == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "label is required")
	}

	disc, err := h.discRepo.Create(c.Request().Context(), pgRepo.CreateDiscParams{
		Label:    req.Label,
		DiscName: req.DiscName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, disc)
}
