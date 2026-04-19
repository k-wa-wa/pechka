package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
)

type SearchHandler struct {
	contentRepo *elasticRepo.ContentRepository
}

func NewSearchHandler(contentRepo *elasticRepo.ContentRepository) *SearchHandler {
	return &SearchHandler{contentRepo: contentRepo}
}

func (h *SearchHandler) Search(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "q parameter is required")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	results, err := h.contentRepo.Search(c.Request().Context(), query, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if results == nil {
		results = []*elasticRepo.SearchResult{}
	}
	return c.JSON(http.StatusOK, results)
}
