package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type pgChannelRepository interface {
	List(ctx context.Context) ([]*domain.Channel, error)
	GetByID(ctx context.Context, id string) (*domain.Channel, error)
	Exists(ctx context.Context, id string) (bool, error)
}

type ChannelsHandler struct {
	channelRepo pgChannelRepository
}

func NewChannelsHandler(channelRepo pgChannelRepository) *ChannelsHandler {
	return &ChannelsHandler{channelRepo: channelRepo}
}

func (h *ChannelsHandler) List(c echo.Context) error {
	channels, err := h.channelRepo.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if channels == nil {
		channels = []*domain.Channel{}
	}
	return c.JSON(http.StatusOK, channels)
}

func (h *ChannelsHandler) Get(c echo.Context) error {
	id := c.Param("id")
	if !domain.IsValidChannelID(id) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel_id")
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "channel not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, channel)
}

// resolveChannelID は X-Channel-Id ヘッダー（またはフォールバック値）からチャンネルIDを決定し、
// フォーマットおよびマスターデータの存在確認を行う。未指定時は 400 Bad Request を返す。
func resolveChannelID(ctx context.Context, c echo.Context, fallback *string, channelRepo pgChannelRepository) (string, error) {
	channelID := strings.TrimSpace(c.Request().Header.Get(domain.HeaderChannelID))
	if channelID == "" && fallback != nil {
		channelID = strings.TrimSpace(*fallback)
	}
	if channelID == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "channel_id is required")
	}

	if !domain.IsValidChannelID(channelID) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid channel_id")
	}

	if channelRepo != nil {
		exists, err := channelRepo.Exists(ctx, channelID)
		if err != nil {
			return "", echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if !exists {
			return "", echo.NewHTTPError(http.StatusBadRequest, "unknown channel_id")
		}
	}

	return channelID, nil
}
