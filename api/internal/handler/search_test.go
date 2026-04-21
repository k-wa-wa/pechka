package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/handler"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
)

type mockElasticContentRepo struct {
	searchFn func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

func (m *mockElasticContentRepo) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	return m.searchFn(ctx, query, limit, offset)
}

func TestSearchHandler_Search_MissingQuery(t *testing.T) {
	h := handler.NewSearchHandler(&mockElasticContentRepo{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Search(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestSearchHandler_Search_DefaultLimit(t *testing.T) {
	mock := &mockElasticContentRepo{
		searchFn: func(_ context.Context, _ string, limit, _ int) ([]*elasticRepo.SearchResult, error) {
			if limit != 20 {
				t.Errorf("expected default limit 20, got %d", limit)
			}
			return []*elasticRepo.SearchResult{}, nil
		},
	}

	h := handler.NewSearchHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Search(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSearchHandler_Search_LimitClamped(t *testing.T) {
	mock := &mockElasticContentRepo{
		searchFn: func(_ context.Context, _ string, limit, _ int) ([]*elasticRepo.SearchResult, error) {
			if limit != 20 {
				t.Errorf("expected clamped limit 20, got %d", limit)
			}
			return []*elasticRepo.SearchResult{}, nil
		},
	}

	h := handler.NewSearchHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test&limit=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Search(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
