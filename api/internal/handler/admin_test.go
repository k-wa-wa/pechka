package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type mockPgContentRepo struct {
	listFn   func(ctx context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error)
	createFn func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
	updateFn func(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockPgContentRepo) List(ctx context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error) {
	return m.listFn(ctx, params)
}
func (m *mockPgContentRepo) Create(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
	return m.createFn(ctx, params)
}
func (m *mockPgContentRepo) Update(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error) {
	return m.updateFn(ctx, params)
}
func (m *mockPgContentRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

type mockPgDiscRepo struct {
	listFn   func(ctx context.Context) ([]*domain.Disc, error)
	createFn func(ctx context.Context, params pgRepo.CreateDiscParams) (*domain.Disc, error)
}

func (m *mockPgDiscRepo) List(ctx context.Context) ([]*domain.Disc, error) {
	return m.listFn(ctx)
}
func (m *mockPgDiscRepo) Create(ctx context.Context, params pgRepo.CreateDiscParams) (*domain.Disc, error) {
	return m.createFn(ctx, params)
}

func newSnowflakeNode(t *testing.T) *snowflake.Node {
	t.Helper()
	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("snowflake: %v", err)
	}
	return node
}

func TestAdminHandler_CreateContent_MissingFields(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"description":"no title or type"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateContent(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_CreateContent_SetsStatusPending(t *testing.T) {
	var gotStatus domain.ContentStatus
	contentRepo := &mockPgContentRepo{
		createFn: func(_ context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
			gotStatus = params.Status
			return &domain.Content{
				ShortID:     params.ShortID,
				ContentType: params.ContentType,
				Title:       params.Title,
				Status:      params.Status,
			}, nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"content_type":"video","title":"Test Video"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if gotStatus != domain.ContentStatusPending {
		t.Errorf("expected status pending, got %q", gotStatus)
	}
}

func TestAdminHandler_ListContents_DefaultLimit(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		listFn: func(_ context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error) {
			if params.Limit != 20 {
				t.Errorf("expected default limit 20, got %d", params.Limit)
			}
			return []*domain.Content{}, nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListContents(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminHandler_UpdateContent_NotFound(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		updateFn: func(_ context.Context, _ pgRepo.UpdateContentParams) (*domain.Content, error) {
			return nil, pgx.ErrNoRows
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"title":"new title"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/contents/999", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := h.UpdateContent(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", he.Code)
	}
}

func TestAdminHandler_CreateDisc_MissingLabel(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/discs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateDisc(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_ListContents_EmptyReturnsArray(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		listFn: func(_ context.Context, _ pgRepo.ListContentsParams) ([]*domain.Content, error) {
			return nil, nil
		},
	}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListContents(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result == nil {
		t.Error("expected empty array, got null")
	}
}
