package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type mongoSubtitleRepository interface {
	GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

type SubtitlesHandler struct {
	subtitleRepo mongoSubtitleRepository
}

func NewSubtitlesHandler(subtitleRepo mongoSubtitleRepository) *SubtitlesHandler {
	return &SubtitlesHandler{subtitleRepo: subtitleRepo}
}

// GetVTT は published の字幕のみを WebVTT として動的に返す。
// draft（未レビュー）の字幕はここに到達する前に Mongo 同期段階で除外されているため、
// このハンドラは公開可否を判定する必要がない。
func (h *SubtitlesHandler) GetVTT(c echo.Context) error {
	shortID := c.Param("short_id")
	language := c.Param("lang")

	sub, err := h.subtitleRepo.GetByShortIDAndLanguage(c.Request().Context(), shortID, language)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return echo.NewHTTPError(http.StatusNotFound, "subtitle not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/vtt; charset=utf-8")
	// 動的生成のため、HLSセグメントとは異なり長期キャッシュしない
	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.String(http.StatusOK, renderVTT(sub.Cues))
}

func renderVTT(cues []domain.MongoCue) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, cue := range cues {
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n", vttTimestamp(cue.StartMs), vttTimestamp(cue.EndMs), cue.Text)
	}
	return b.String()
}

func vttTimestamp(ms int) string {
	h := ms / 3600000
	m := (ms % 3600000) / 60000
	s := (ms % 60000) / 1000
	msRem := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, msRem)
}
