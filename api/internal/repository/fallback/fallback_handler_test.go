package fallback_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	"github.com/k-wa-wa/pechka/api/internal/repository/fallback"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
)

// Acceptance Condition 1: When MongoDB is down, GET /contents, GET /contents/:short_id,
// GET /contents/:short_id/variants, and GET /contents/:short_id/subtitles/:lang return 200 OK with full data.
func TestAcceptance_MongoDBDown_HandlersReturn200OK(t *testing.T) {
	e := echo.New()

	// Primary Mongo repos that always fail (simulating node crash / connection timeout)
	failingPrimaryContent := &mockPrimaryContentRepo{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return nil, errors.New("mongo node is down: connection refused")
		},
		getByShortIDFn: func(ctx context.Context, shortID string) (*domain.MongoContent, error) {
			return nil, errors.New("mongo node is down: connection refused")
		},
		getVariantsFn: func(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
			return nil, errors.New("mongo node is down: connection refused")
		},
	}
	failingPrimarySubtitle := &mockPrimarySubtitleRepo{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return nil, errors.New("mongo node is down: connection refused")
		},
	}

	// Fallback PostgreSQL stores returning valid data
	fbContentStore := &mockFallbackContentStore{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return []*domain.MongoContent{
				{
					ID:          "short-1",
					Title:       "Postgres Content",
					ContentType: domain.ContentTypeVideo,
					ChannelID:   "channel-a",
					ChannelName: "Channel Alpha",
					Status:      domain.ContentStatusReady,
					Tags:        []string{"anime"},
					Variants: []domain.MongoVariant{
						{VariantType: "hls", HLSKey: "resources/video/short-1/master.m3u8"},
					},
					HasSubtitles: true,
				},
			}, nil
		},
		getByShortIDFn: func(ctx context.Context, shortID string) (*domain.MongoContent, error) {
			return &domain.MongoContent{
				ID:          shortID,
				Title:       "Postgres Content " + shortID,
				ContentType: domain.ContentTypeVideo,
				ChannelID:   "channel-a",
				ChannelName: "Channel Alpha",
				Status:      domain.ContentStatusReady,
				Tags:        []string{"anime"},
				Variants: []domain.MongoVariant{
					{VariantType: "hls", HLSKey: "resources/video/" + shortID + "/master.m3u8"},
				},
				HasSubtitles: true,
			}, nil
		},
		getVariantsFn: func(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
			return []domain.MongoVariant{
				{VariantType: "hls", HLSKey: "resources/video/" + shortID + "/master.m3u8"},
			}, nil
		},
	}

	fbSubtitleStore := &mockFallbackSubtitleStore{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return &domain.MongoSubtitle{
				ID:       shortID + ":" + language,
				ShortID:  shortID,
				Language: language,
				Status:   "published",
				Cues: []domain.MongoCue{
					{StartMs: 0, EndMs: 2500, Text: "こんにちは、世界！"},
				},
			}, nil
		},
	}

	mongoBreaker := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 50 * time.Millisecond,
	})

	fbContentRepo := fallback.NewContentRepositoryWithStore(failingPrimaryContent, fbContentStore, mongoBreaker)
	fbSubtitleRepo := fallback.NewSubtitleRepositoryWithStore(failingPrimarySubtitle, fbSubtitleStore, mongoBreaker)

	contentsHandler := handler.NewContentsHandler(fbContentRepo)
	subtitlesHandler := handler.NewSubtitlesHandler(fbSubtitleRepo)

	// 1. GET /api/v1/contents
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := contentsHandler.List(c); err != nil {
			t.Fatalf("contentsHandler.List failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var list []*domain.MongoContent
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(list) != 1 || list[0].Title != "Postgres Content" {
			t.Errorf("unexpected content in list: %+v", list)
		}
	}

	// 2. GET /api/v1/contents/:short_id
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short-1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("short_id")
		c.SetParamValues("short-1")
		if err := contentsHandler.Get(c); err != nil {
			t.Fatalf("contentsHandler.Get failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var content domain.MongoContent
		if err := json.Unmarshal(rec.Body.Bytes(), &content); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if content.ID != "short-1" || content.Title != "Postgres Content short-1" {
			t.Errorf("unexpected content details: %+v", content)
		}
	}

	// 3. GET /api/v1/contents/:short_id/variants
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short-1/variants", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("short_id")
		c.SetParamValues("short-1")
		if err := contentsHandler.GetVariants(c); err != nil {
			t.Fatalf("contentsHandler.GetVariants failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var variants []domain.MongoVariant
		if err := json.Unmarshal(rec.Body.Bytes(), &variants); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(variants) != 1 || variants[0].VariantType != "hls" {
			t.Errorf("unexpected variants: %+v", variants)
		}
	}

	// 4. GET /api/v1/contents/:short_id/subtitles/:lang
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short-1/subtitles/ja", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("short_id", "lang")
		c.SetParamValues("short-1", "ja")
		if err := subtitlesHandler.GetVTT(c); err != nil {
			t.Fatalf("subtitlesHandler.GetVTT failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		body := rec.Body.String()
		if rec.Header().Get(echo.HeaderContentType) != "text/vtt; charset=utf-8" {
			t.Errorf("expected text/vtt content-type, got %s", rec.Header().Get(echo.HeaderContentType))
		}
		if body == "" || !jsonValidSubstring(body, "WEBVTT") || !jsonValidSubstring(body, "こんにちは、世界！") {
			t.Errorf("unexpected VTT response: %s", body)
		}
	}
}

// Acceptance Condition 2: When Elasticsearch is down, GET /search?q=... returns 200 OK with search results.
func TestAcceptance_ElasticsearchDown_SearchHandlerReturns200OK(t *testing.T) {
	e := echo.New()

	failingPrimarySearch := &mockPrimarySearchRepo{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			return nil, errors.New("elasticsearch connection refused")
		},
	}

	fbSearchStore := &mockFallbackSearchStore{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			return []*elasticRepo.SearchResult{
				{
					ShortID:     "search-1",
					Title:       "Found via Postgres ILIKE",
					Description: "Matching description for " + query,
					ContentType: domain.ContentTypeVideo,
					ChannelID:   "channel-1",
					ChannelName: "Channel One",
					Tags:        []string{"search"},
					Status:      domain.ContentStatusReady,
				},
			}, nil
		},
	}

	elasticBreaker := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "elastic",
		Timeout: 50 * time.Millisecond,
	})

	fbSearchRepo := fallback.NewSearchRepositoryWithStore(failingPrimarySearch, fbSearchStore, elasticBreaker)
	searchHandler := handler.NewSearchHandler(fbSearchRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=golang", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := searchHandler.Search(c); err != nil {
		t.Fatalf("searchHandler.Search failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var results []*elasticRepo.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Found via Postgres ILIKE" {
		t.Errorf("unexpected search results: %+v", results)
	}
}

func jsonValidSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
