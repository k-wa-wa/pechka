package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type pgContentRepository interface {
	List(ctx context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error)
	Count(ctx context.Context, params pgRepo.ListContentsParams) (int, error)
	Create(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
	Update(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error)
	Delete(ctx context.Context, id string) error
	SetArchived(ctx context.Context, id string, archived bool) (*domain.Content, error)
}

type pgDiscRepository interface {
	List(ctx context.Context) ([]*domain.Disc, error)
	Create(ctx context.Context, params pgRepo.CreateDiscParams) (*domain.Disc, error)
}

type pgSubtitleRepository interface {
	ListTracksByContentID(ctx context.Context, contentID string) ([]*domain.SubtitleTrack, error)
	GetTrack(ctx context.Context, trackID string) (*domain.SubtitleTrack, error)
	SetTrackStatus(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error)
	ListCuesByTrackID(ctx context.Context, trackID string) ([]*domain.SubtitleCue, error)
	UpdateCue(ctx context.Context, params pgRepo.UpdateCueParams) (*domain.SubtitleCue, error)
	InsertCue(ctx context.Context, params pgRepo.InsertCueParams) (*domain.SubtitleCue, error)
	DeleteCue(ctx context.Context, id string) error
}

type AdminHandler struct {
	contentRepo  pgContentRepository
	discRepo     pgDiscRepository
	subtitleRepo pgSubtitleRepository
	channelRepo  pgChannelRepository
	snowflake    *snowflake.Node
}

func NewAdminHandler(contentRepo pgContentRepository, discRepo pgDiscRepository, subtitleRepo pgSubtitleRepository, node *snowflake.Node) *AdminHandler {
	return &AdminHandler{contentRepo: contentRepo, discRepo: discRepo, subtitleRepo: subtitleRepo, snowflake: node}
}

func (h *AdminHandler) WithChannelRepository(channelRepo pgChannelRepository) *AdminHandler {
	h.channelRepo = channelRepo
	return h
}

type createContentRequest struct {
	ContentType      domain.ContentType    `json:"content_type" validate:"required"`
	ChannelID        *string               `json:"channel_id"`
	DiscID           *string               `json:"disc_id"`
	Title            string                `json:"title" validate:"required"`
	Description      string                `json:"description"`
	DurationSeconds  *int                  `json:"duration_seconds"`
	Is360            bool                  `json:"is_360"`
	Tags             []string              `json:"tags"`
	DocumentKey      *string               `json:"document_key"`
	DocumentMimeType *string               `json:"document_mime_type"`
	Status           *domain.ContentStatus `json:"status"`
}

func (h *AdminHandler) CreateContent(c echo.Context) error {
	var req createContentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Title == "" || req.ContentType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and content_type are required")
	}
	if req.DocumentMimeType != nil && *req.DocumentMimeType != "" && !domain.IsSupportedDocumentMimeType(*req.DocumentMimeType) {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported document_mime_type")
	}

	ctx := c.Request().Context()
	channelID, err := resolveChannelID(ctx, c, req.ChannelID, h.channelRepo)
	if err != nil {
		return err
	}

	shortID := h.snowflake.Generate().String()
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	status := domain.ContentStatusPending
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	content, err := h.contentRepo.Create(ctx, pgRepo.CreateContentParams{
		ShortID:          shortID,
		ContentType:      req.ContentType,
		ChannelID:        channelID,
		DiscID:           req.DiscID,
		Title:            req.Title,
		Description:      req.Description,
		DurationSeconds:  req.DurationSeconds,
		Is360:            req.Is360,
		Tags:             tags,
		Status:           status,
		DocumentKey:      req.DocumentKey,
		DocumentMimeType: req.DocumentMimeType,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, content)
}

type updateContentRequest struct {
	ChannelID   *string               `json:"channel_id"`
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

	ctx := c.Request().Context()
	var channelID *string
	if headerCh := strings.TrimSpace(c.Request().Header.Get(domain.HeaderChannelID)); headerCh != "" {
		channelID = &headerCh
	} else if req.ChannelID != nil {
		trimmed := strings.TrimSpace(*req.ChannelID)
		channelID = &trimmed
	}

	if channelID != nil {
		validated, err := resolveChannelID(ctx, c, channelID, h.channelRepo)
		if err != nil {
			return err
		}
		channelID = &validated
	}

	content, err := h.contentRepo.Update(ctx, pgRepo.UpdateContentParams{
		ID:          id,
		ChannelID:   channelID,
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

// ArchiveContent はコンテンツを論理削除(アーカイブ)する。
// 物理データは削除せず archived_at を設定し、admin画面以外(公開API/検索)から見えなくする。
func (h *AdminHandler) ArchiveContent(c echo.Context) error {
	id := c.Param("id")
	content, err := h.contentRepo.SetArchived(c.Request().Context(), id, true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "content not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, content)
}

// UnarchiveContent はアーカイブ済みコンテンツを元の公開状態に戻す。
func (h *AdminHandler) UnarchiveContent(c echo.Context) error {
	id := c.Param("id")
	content, err := h.contentRepo.SetArchived(c.Request().Context(), id, false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "content not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, content)
}

type listContentsResponse struct {
	Contents []*domain.Content `json:"contents"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
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

	var channelID *string
	if ch := c.QueryParam("channel_id"); ch != "" {
		if !domain.IsValidChannelID(ch) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid channel_id")
		}
		channelID = &ch
	}

	listParams := pgRepo.ListContentsParams{
		Status:    status,
		ChannelID: channelID,
		// admin画面ではアーカイブ済みコンテンツも一覧・管理できるようにする。
		IncludeArchived: true,
		Limit:           limit,
		Offset:          offset,
	}

	contents, err := h.contentRepo.List(c.Request().Context(), listParams)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	total, err := h.contentRepo.Count(c.Request().Context(), listParams)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if contents == nil {
		contents = []*domain.Content{}
	}
	return c.JSON(http.StatusOK, listContentsResponse{
		Contents: contents,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
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

func (h *AdminHandler) ListSubtitleTracks(c echo.Context) error {
	contentID := c.Param("content_id")
	tracks, err := h.subtitleRepo.ListTracksByContentID(c.Request().Context(), contentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if tracks == nil {
		tracks = []*domain.SubtitleTrack{}
	}
	return c.JSON(http.StatusOK, tracks)
}

func (h *AdminHandler) ListSubtitleCues(c echo.Context) error {
	trackID := c.Param("track_id")
	cues, err := h.subtitleRepo.ListCuesByTrackID(c.Request().Context(), trackID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if cues == nil {
		cues = []*domain.SubtitleCue{}
	}
	return c.JSON(http.StatusOK, cues)
}

type updateSubtitleTrackStatusRequest struct {
	Status domain.SubtitleTrackStatus `json:"status" validate:"required"`
}

func (h *AdminHandler) UpdateSubtitleTrackStatus(c echo.Context) error {
	trackID := c.Param("track_id")
	var req updateSubtitleTrackStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Status != domain.SubtitleTrackStatusDraft && req.Status != domain.SubtitleTrackStatusPublished {
		return echo.NewHTTPError(http.StatusBadRequest, "status must be draft or published")
	}

	track, err := h.subtitleRepo.SetTrackStatus(c.Request().Context(), trackID, req.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "subtitle track not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, track)
}

type updateSubtitleCueRequest struct {
	Text    *string `json:"text"`
	StartMs *int    `json:"start_ms"`
	EndMs   *int    `json:"end_ms"`
}

func (h *AdminHandler) UpdateSubtitleCue(c echo.Context) error {
	cueID := c.Param("cue_id")
	var req updateSubtitleCueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cue, err := h.subtitleRepo.UpdateCue(c.Request().Context(), pgRepo.UpdateCueParams{
		ID:      cueID,
		Text:    req.Text,
		StartMs: req.StartMs,
		EndMs:   req.EndMs,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "cue not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cue)
}

type insertSubtitleCueRequest struct {
	Seq     int    `json:"seq"`
	StartMs int    `json:"start_ms"`
	EndMs   int    `json:"end_ms"`
	Text    string `json:"text"`
}

func (h *AdminHandler) InsertSubtitleCue(c echo.Context) error {
	trackID := c.Param("track_id")
	var req insertSubtitleCueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cue, err := h.subtitleRepo.InsertCue(c.Request().Context(), pgRepo.InsertCueParams{
		TrackID: trackID,
		Seq:     req.Seq,
		StartMs: req.StartMs,
		EndMs:   req.EndMs,
		Text:    req.Text,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, cue)
}

func (h *AdminHandler) DeleteSubtitleCue(c echo.Context) error {
	cueID := c.Param("cue_id")
	if err := h.subtitleRepo.DeleteCue(c.Request().Context(), cueID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
