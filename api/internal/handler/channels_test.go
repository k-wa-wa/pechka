package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type mockPgChannelRepo struct {
	listFn    func(ctx context.Context) ([]*domain.Channel, error)
	getByIDFn func(ctx context.Context, id string) (*domain.Channel, error)
	existsFn  func(ctx context.Context, id string) (bool, error)
}

func (m *mockPgChannelRepo) List(ctx context.Context) ([]*domain.Channel, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx)
}

func (m *mockPgChannelRepo) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	if m.getByIDFn == nil {
		return nil, pgx.ErrNoRows
	}
	return m.getByIDFn(ctx, id)
}

func (m *mockPgChannelRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.existsFn == nil {
		return true, nil
	}
	return m.existsFn(ctx, id)
}

func TestChannelsHandler_List_Success(t *testing.T) {
	repo := &mockPgChannelRepo{
		listFn: func(_ context.Context) ([]*domain.Channel, error) {
			return []*domain.Channel{
				{ID: "tech-feed", Name: "Tech Feed"},
				{ID: "bluray", Name: "Blu-ray"},
			}, nil
		},
	}

	h := handler.NewChannelsHandler(repo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/channels", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var channels []domain.Channel
	if err := json.Unmarshal(rec.Body.Bytes(), &channels); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

func TestChannelsHandler_List_EmptyReturnsArray(t *testing.T) {
	repo := &mockPgChannelRepo{
		listFn: func(_ context.Context) ([]*domain.Channel, error) {
			return nil, nil
		},
	}

	h := handler.NewChannelsHandler(repo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/channels", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var channels []domain.Channel
	if err := json.Unmarshal(rec.Body.Bytes(), &channels); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if channels == nil || len(channels) != 0 {
		t.Errorf("expected empty array, got %v", channels)
	}
}

func TestChannelsHandler_Get_Success(t *testing.T) {
	repo := &mockPgChannelRepo{
		getByIDFn: func(_ context.Context, id string) (*domain.Channel, error) {
			return &domain.Channel{ID: id, Name: "Blu-ray", Description: "Blu-ray取り込み動画"}, nil
		},
	}

	h := handler.NewChannelsHandler(repo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/channels/bluray", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bluray")

	if err := h.Get(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var ch domain.Channel
	if err := json.Unmarshal(rec.Body.Bytes(), &ch); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ch.ID != "bluray" || ch.Name != "Blu-ray" {
		t.Errorf("unexpected channel: %+v", ch)
	}
}

func TestChannelsHandler_Get_InvalidFormat(t *testing.T) {
	h := handler.NewChannelsHandler(&mockPgChannelRepo{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/channels/INVALID_ID!", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("INVALID_ID!")

	err := h.Get(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestChannelsHandler_Get_NotFound(t *testing.T) {
	repo := &mockPgChannelRepo{
		getByIDFn: func(_ context.Context, _ string) (*domain.Channel, error) {
			return nil, pgx.ErrNoRows
		},
	}

	h := handler.NewChannelsHandler(repo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/channels/unknown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("unknown")

	err := h.Get(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", he.Code)
	}
}

func TestAdminHandler_CreateContent_WithXChannelIdHeader(t *testing.T) {
	var gotChannelID string
	contentRepo := &mockPgContentRepo{
		createFn: func(_ context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
			gotChannelID = params.ChannelID
			return &domain.Content{
				ShortID:     params.ShortID,
				ContentType: params.ContentType,
				ChannelID:   params.ChannelID,
				Title:       params.Title,
				Status:      params.Status,
			}, nil
		},
	}
	channelRepo := &mockPgChannelRepo{
		existsFn: func(_ context.Context, id string) (bool, error) {
			return id == "tech-feed" || id == "bluray", nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).WithChannelRepository(channelRepo)
	e := echo.New()

	body := `{"content_type":"video","title":"Tech Digest"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "tech-feed")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotChannelID != "tech-feed" {
		t.Errorf("expected channel_id tech-feed, got %q", gotChannelID)
	}
}

func TestAdminHandler_CreateContent_InvalidXChannelId(t *testing.T) {
	channelRepo := &mockPgChannelRepo{
		existsFn: func(_ context.Context, id string) (bool, error) {
			return id == "tech-feed" || id == "bluray", nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).WithChannelRepository(channelRepo)
	e := echo.New()

	body := `{"content_type":"video","title":"Tech Digest"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "nonexistent-channel")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateContent(c)
	if err == nil {
		t.Fatal("expected error for unknown channel, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_CreateContent_MissingChannelId(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"content_type":"video","title":"Tech Digest"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateContent(c)
	if err == nil {
		t.Fatal("expected error when channel_id is missing, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_ListContents_ChannelFilter(t *testing.T) {
	var gotChannelID *string
	contentRepo := &mockPgContentRepo{
		listFn: func(_ context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error) {
			gotChannelID = params.ChannelID
			return []*domain.Content{}, nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?channel_id=tech-feed", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListContents(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotChannelID == nil || *gotChannelID != "tech-feed" {
		t.Errorf("expected channel_id tech-feed, got %v", gotChannelID)
	}
}

func TestAdminHandler_ListContents_InvalidChannelFilter(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?channel_id=INVALID!", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListContents(c)
	if err == nil {
		t.Fatal("expected error for invalid channel_id filter, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_UpdateContent_WithChannelId(t *testing.T) {
	var gotChannelID *string
	contentRepo := &mockPgContentRepo{
		updateFn: func(_ context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error) {
			gotChannelID = params.ChannelID
			return &domain.Content{ID: params.ID, ChannelID: *params.ChannelID}, nil
		},
	}
	channelRepo := &mockPgChannelRepo{
		existsFn: func(_ context.Context, id string) (bool, error) {
			return id == "tech-feed" || id == "bluray", nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).WithChannelRepository(channelRepo)
	e := echo.New()

	req := httptest.NewRequest(http.MethodPut, "/v1/admin/contents/10", strings.NewReader(`{"channel_id":"tech-feed"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("10")

	if err := h.UpdateContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotChannelID == nil || *gotChannelID != "tech-feed" {
		t.Errorf("expected channel_id tech-feed, got %v", gotChannelID)
	}
}

func TestAdminHandler_UpdateContent_EmptyChannelIdRejected(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()

	req := httptest.NewRequest(http.MethodPut, "/v1/admin/contents/10", strings.NewReader(`{"channel_id":""}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("10")

	err := h.UpdateContent(c)
	if err == nil {
		t.Fatal("expected error for empty channel_id, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}
