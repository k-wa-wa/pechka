package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
)

type mockMongoContentRepo struct {
	listFn             func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	getByShortIDFn     func(ctx context.Context, shortID string) (*domain.MongoContent, error)
	getVariantsFn      func(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
}

func (m *mockMongoContentRepo) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	return m.listFn(ctx, params)
}

func (m *mockMongoContentRepo) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	return m.getByShortIDFn(ctx, shortID)
}

func (m *mockMongoContentRepo) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	return m.getVariantsFn(ctx, shortID)
}

func TestContentsHandler_List_DefaultLimit(t *testing.T) {
	mock := &mockMongoContentRepo{
		listFn: func(_ context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			if params.Limit != 20 {
				t.Errorf("expected default limit 20, got %d", params.Limit)
			}
			return []*domain.MongoContent{}, nil
		},
	}

	h := handler.NewContentsHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestContentsHandler_List_LimitClamped(t *testing.T) {
	mock := &mockMongoContentRepo{
		listFn: func(_ context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			if params.Limit != 20 {
				t.Errorf("expected clamped limit 20, got %d", params.Limit)
			}
			return []*domain.MongoContent{}, nil
		},
	}

	h := handler.NewContentsHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents?limit=999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContentsHandler_List_EmptyReturnsArray(t *testing.T) {
	mock := &mockMongoContentRepo{
		listFn: func(_ context.Context, _ mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return nil, nil
		},
	}

	h := handler.NewContentsHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
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

func TestContentsHandler_Get_NotFound(t *testing.T) {
	mock := &mockMongoContentRepo{
		getByShortIDFn: func(_ context.Context, _ string) (*domain.MongoContent, error) {
			return nil, echo.ErrNotFound
		},
	}

	h := handler.NewContentsHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents/no-such-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("short_id")
	c.SetParamValues("no-such-id")

	err := h.Get(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if he.Code != http.StatusNotFound && he.Code != http.StatusInternalServerError {
		t.Errorf("unexpected status: %d", he.Code)
	}
}
