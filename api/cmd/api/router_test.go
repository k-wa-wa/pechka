package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type mockChannelRepo struct {
	listFn    func(ctx context.Context) ([]*domain.Channel, error)
	getByIDFn func(ctx context.Context, id string) (*domain.Channel, error)
	existsFn  func(ctx context.Context, id string) (bool, error)
}

func (m *mockChannelRepo) List(ctx context.Context) ([]*domain.Channel, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*domain.Channel{}, nil
}

func (m *mockChannelRepo) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, pgx.ErrNoRows
}

func (m *mockChannelRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	return false, nil
}

type mockMongoContentRepo struct {
	listFn         func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	getByShortIDFn func(ctx context.Context, shortID string) (*domain.MongoContent, error)
	getVariantsFn  func(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
}

func (m *mockMongoContentRepo) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return []*domain.MongoContent{}, nil
}

func (m *mockMongoContentRepo) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	if m.getByShortIDFn != nil {
		return m.getByShortIDFn(ctx, shortID)
	}
	return nil, mongo.ErrNoDocuments
}

func (m *mockMongoContentRepo) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	if m.getVariantsFn != nil {
		return m.getVariantsFn(ctx, shortID)
	}
	return []domain.MongoVariant{}, nil
}

type mockMongoSubtitleRepo struct {
	getByShortIDAndLangFn func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

func (m *mockMongoSubtitleRepo) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	if m.getByShortIDAndLangFn != nil {
		return m.getByShortIDAndLangFn(ctx, shortID, language)
	}
	return nil, mongo.ErrNoDocuments
}

type mockElasticContentRepo struct {
	searchFn func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

func (m *mockElasticContentRepo) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query, limit, offset)
	}
	return []*elasticRepo.SearchResult{}, nil
}

type mockPgContentRepo struct {
	listFn        func(ctx context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error)
	countFn       func(ctx context.Context, params pgRepo.ListContentsParams) (int, error)
	createFn      func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
	updateFn      func(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error)
	deleteFn      func(ctx context.Context, id string) error
	setArchivedFn func(ctx context.Context, id string, archived bool) (*domain.Content, error)
}

func (m *mockPgContentRepo) List(ctx context.Context, params pgRepo.ListContentsParams) ([]*domain.Content, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return []*domain.Content{}, nil
}
func (m *mockPgContentRepo) Count(ctx context.Context, params pgRepo.ListContentsParams) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, params)
	}
	return 0, nil
}
func (m *mockPgContentRepo) Create(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return nil, nil
}
func (m *mockPgContentRepo) Update(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, params)
	}
	return nil, nil
}
func (m *mockPgContentRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockPgContentRepo) SetArchived(ctx context.Context, id string, archived bool) (*domain.Content, error) {
	if m.setArchivedFn != nil {
		return m.setArchivedFn(ctx, id, archived)
	}
	return nil, nil
}

type mockPgDiscRepo struct {
	listFn   func(ctx context.Context) ([]*domain.Disc, error)
	createFn func(ctx context.Context, params pgRepo.CreateDiscParams) (*domain.Disc, error)
}

func (m *mockPgDiscRepo) List(ctx context.Context) ([]*domain.Disc, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*domain.Disc{}, nil
}
func (m *mockPgDiscRepo) Create(ctx context.Context, params pgRepo.CreateDiscParams) (*domain.Disc, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return nil, nil
}

type mockPgSubtitleRepo struct {
	listTracksFn func(ctx context.Context, contentID string) ([]*domain.SubtitleTrack, error)
	getTrackFn   func(ctx context.Context, trackID string) (*domain.SubtitleTrack, error)
	setStatusFn  func(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error)
	listCuesFn   func(ctx context.Context, trackID string) ([]*domain.SubtitleCue, error)
	updateCueFn  func(ctx context.Context, params pgRepo.UpdateCueParams) (*domain.SubtitleCue, error)
	insertCueFn  func(ctx context.Context, params pgRepo.InsertCueParams) (*domain.SubtitleCue, error)
	deleteCueFn  func(ctx context.Context, id string) error
}

func (m *mockPgSubtitleRepo) ListTracksByContentID(ctx context.Context, contentID string) ([]*domain.SubtitleTrack, error) {
	if m.listTracksFn != nil {
		return m.listTracksFn(ctx, contentID)
	}
	return []*domain.SubtitleTrack{}, nil
}
func (m *mockPgSubtitleRepo) GetTrack(ctx context.Context, trackID string) (*domain.SubtitleTrack, error) {
	if m.getTrackFn != nil {
		return m.getTrackFn(ctx, trackID)
	}
	return nil, nil
}
func (m *mockPgSubtitleRepo) SetTrackStatus(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(ctx, trackID, status)
	}
	return nil, nil
}
func (m *mockPgSubtitleRepo) ListCuesByTrackID(ctx context.Context, trackID string) ([]*domain.SubtitleCue, error) {
	if m.listCuesFn != nil {
		return m.listCuesFn(ctx, trackID)
	}
	return []*domain.SubtitleCue{}, nil
}
func (m *mockPgSubtitleRepo) UpdateCue(ctx context.Context, params pgRepo.UpdateCueParams) (*domain.SubtitleCue, error) {
	if m.updateCueFn != nil {
		return m.updateCueFn(ctx, params)
	}
	return nil, nil
}
func (m *mockPgSubtitleRepo) InsertCue(ctx context.Context, params pgRepo.InsertCueParams) (*domain.SubtitleCue, error) {
	if m.insertCueFn != nil {
		return m.insertCueFn(ctx, params)
	}
	return nil, nil
}
func (m *mockPgSubtitleRepo) DeleteCue(ctx context.Context, id string) error {
	if m.deleteCueFn != nil {
		return m.deleteCueFn(ctx, id)
	}
	return nil
}

func setupTestRouter(allowedIPRange string) (*RouterParams, *mockChannelRepo, *mockMongoContentRepo, *mockPgContentRepo, *mockMongoSubtitleRepo, *mockElasticContentRepo) {
	chRepo := &mockChannelRepo{}
	mgRepo := &mockMongoContentRepo{}
	pgContRepo := &mockPgContentRepo{}
	pgDisc := &mockPgDiscRepo{}
	pgSub := &mockPgSubtitleRepo{}
	mgSub := &mockMongoSubtitleRepo{}
	esRepo := &mockElasticContentRepo{}

	sfNode, _ := snowflake.NewNode(1)
	contentsH := handler.NewContentsHandler(mgRepo)
	channelsH := handler.NewChannelsHandler(chRepo)
	subtitlesH := handler.NewSubtitlesHandler(mgSub)
	searchH := handler.NewSearchHandler(esRepo)
	adminH := handler.NewAdminHandler(pgContRepo, pgDisc, pgSub, sfNode).WithChannelRepository(chRepo)
	ingestH := handler.NewIngestHandler(nil, "test-ns").WithChannelRepository(chRepo)
	uploadH := handler.NewUploadHandler(pgContRepo, nil, "test-bucket", nil, sfNode, "test-ns").WithChannelRepository(chRepo)

	reg := prometheus.NewRegistry()

	params := &RouterParams{
		AllowedIPRange:    allowedIPRange,
		MetricsRegisterer: reg,
		MetricsGatherer:   reg,
		ContentsHandler:   contentsH,
		ChannelsHandler:   channelsH,
		SubtitlesHandler:  subtitlesH,
		SearchHandler:     searchH,
		AdminHandler:      adminH,
		IngestHandler:     ingestH,
		UploadHandler:     uploadH,
	}

	return params, chRepo, mgRepo, pgContRepo, mgSub, esRepo
}

func TestSetupRouter_PanicsOnMissingHandler(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("")

	t.Run("Panics when ChannelsHandler is nil", func(t *testing.T) {
		p := *params
		p.ChannelsHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when ContentsHandler is nil", func(t *testing.T) {
		p := *params
		p.ContentsHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when SubtitlesHandler is nil", func(t *testing.T) {
		p := *params
		p.SubtitlesHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when SearchHandler is nil", func(t *testing.T) {
		p := *params
		p.SearchHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when AdminHandler is nil", func(t *testing.T) {
		p := *params
		p.AdminHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when IngestHandler is nil", func(t *testing.T) {
		p := *params
		p.IngestHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})

	t.Run("Panics when UploadHandler is nil", func(t *testing.T) {
		p := *params
		p.UploadHandler = nil
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()
		_ = setupRouter(p)
	})
}

func TestRouter_Health(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", res["status"])
	}
}

func TestRouter_Metrics(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "promhttp_metric_handler_requests_total") &&
		!strings.Contains(rec.Body.String(), "pechka_api") {
		t.Errorf("expected prometheus metrics in response body, got: %s", rec.Body.String())
	}
}

func TestRouter_Channels(t *testing.T) {
	params, chRepo, _, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("GET /api/v1/channels - List Channels", func(t *testing.T) {
		chRepo.listFn = func(_ context.Context) ([]*domain.Channel, error) {
			return []*domain.Channel{
				{ID: "ch-1", Name: "Channel 1"},
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/channels", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var channels []*domain.Channel
		if err := json.Unmarshal(rec.Body.Bytes(), &channels); err != nil {
			t.Fatalf("failed to unmarshal channels: %v", err)
		}
		if len(channels) != 1 || channels[0].ID != "ch-1" {
			t.Errorf("unexpected channels: %+v", channels)
		}
	})

	t.Run("GET /api/v1/channels/:id - Found", func(t *testing.T) {
		chRepo.getByIDFn = func(_ context.Context, id string) (*domain.Channel, error) {
			return &domain.Channel{ID: id, Name: "Test Channel"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/valid-channel-id", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/channels/:id - Invalid ID (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/invalid!id@with#bad$chars", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/channels/:id - Not Found (404)", func(t *testing.T) {
		chRepo.getByIDFn = func(_ context.Context, id string) (*domain.Channel, error) {
			return nil, pgx.ErrNoRows
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/nonexistent-channel", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestRouter_Contents(t *testing.T) {
	params, _, mgRepo, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("GET /api/v1/contents - List", func(t *testing.T) {
		mgRepo.listFn = func(_ context.Context, _ mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return []*domain.MongoContent{
				{ID: "short1", Title: "Content 1"},
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var contents []*domain.MongoContent
		if err := json.Unmarshal(rec.Body.Bytes(), &contents); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(contents) != 1 || contents[0].ID != "short1" {
			t.Errorf("unexpected content list: %+v", contents)
		}
	})

	t.Run("GET /api/v1/contents/:short_id - Found", func(t *testing.T) {
		mgRepo.getByShortIDFn = func(_ context.Context, shortID string) (*domain.MongoContent, error) {
			return &domain.MongoContent{ID: shortID, Title: "Detail Content"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short123", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/contents/:short_id - Not Found (404)", func(t *testing.T) {
		mgRepo.getByShortIDFn = func(_ context.Context, _ string) (*domain.MongoContent, error) {
			return nil, mongo.ErrNoDocuments
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/not-found", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/contents/:short_id/variants - Found", func(t *testing.T) {
		mgRepo.getVariantsFn = func(_ context.Context, shortID string) ([]domain.MongoVariant, error) {
			return []domain.MongoVariant{
				{VariantType: "720p", HLSKey: "res/720p.m3u8"},
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short123/variants", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var variants []domain.MongoVariant
		if err := json.Unmarshal(rec.Body.Bytes(), &variants); err != nil {
			t.Fatalf("failed to unmarshal variants: %v", err)
		}
		if len(variants) != 1 || variants[0].VariantType != "720p" {
			t.Errorf("unexpected variants: %+v", variants)
		}
	})
}

func TestRouter_Subtitles(t *testing.T) {
	params, _, _, _, mgSub, _ := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("GET /api/v1/contents/:short_id/subtitles/:lang - Found", func(t *testing.T) {
		mgSub.getByShortIDAndLangFn = func(_ context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return &domain.MongoSubtitle{
				ShortID:  shortID,
				Language: language,
				Cues: []domain.MongoCue{
					{
						StartMs: 1000,
						EndMs:   4000,
						Text:    "Hello world",
					},
				},
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short123/subtitles/ja", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "WEBVTT") || !strings.Contains(rec.Body.String(), "Hello world") {
			t.Errorf("unexpected vtt body: %s", rec.Body.String())
		}
	})

	t.Run("GET /api/v1/contents/:short_id/subtitles/:lang - Not Found (404)", func(t *testing.T) {
		mgSub.getByShortIDAndLangFn = func(_ context.Context, _, _ string) (*domain.MongoSubtitle, error) {
			return nil, mongo.ErrNoDocuments
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/short123/subtitles/en", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestRouter_Search(t *testing.T) {
	params, _, _, _, _, esRepo := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("GET /api/v1/search - With query", func(t *testing.T) {
		esRepo.searchFn = func(_ context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			return []*elasticRepo.SearchResult{
				{ShortID: "c1", Title: "Result 1"},
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var results []*elasticRepo.SearchResult
		if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
			t.Fatalf("failed to unmarshal search results: %v", err)
		}
		if len(results) != 1 || results[0].Title != "Result 1" {
			t.Errorf("unexpected results: %+v", results)
		}
	})

	t.Run("GET /api/v1/search - Missing query (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestRouter_Ingest_And_Upload_Routing(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("192.168.1.0/24")
	router := setupRouter(*params)

	t.Run("POST /api/v1/contents/ingest - Reaches handler and rejects empty body (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/ingest", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.50:54321"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /api/v1/admin/contents/upload - Reaches handler and rejects non-multipart (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/contents/upload", strings.NewReader("bad"))
		req.RemoteAddr = "192.168.1.50:54321"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestRouter_IPFilter(t *testing.T) {
	params, _, _, pgContRepo, _, _ := setupTestRouter("192.168.1.0/24")
	pgContRepo.listFn = func(_ context.Context, _ pgRepo.ListContentsParams) ([]*domain.Content, error) {
		return []*domain.Content{}, nil
	}
	router := setupRouter(*params)

	t.Run("Allowed IP passes to admin endpoint via RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		req.RemoteAddr = "192.168.1.50:54321"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Allowed IP passes to admin endpoint via X-Real-IP header from trusted reverse proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		req.RemoteAddr = "127.0.0.1:54321"
		req.Header.Set("X-Real-IP", "192.168.1.50")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Disallowed IP is blocked with 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		req.RemoteAddr = "10.0.0.1:54321"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Spoofed X-Real-IP from untrusted direct peer is rejected with 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		// 203.0.113.5 is an untrusted public IP (RFC 5737 TEST-NET-3).
		req.RemoteAddr = "203.0.113.5:54321"
		req.Header.Set("X-Real-IP", "192.168.1.50")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: spoofed X-Real-IP must not be trusted from external peer", rec.Code)
		}
	})

	t.Run("Spoofed X-Forwarded-For from untrusted direct peer is ignored and rejected with 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		req.RemoteAddr = "203.0.113.5:54321"
		req.Header.Set("X-Forwarded-For", "192.168.1.50")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: spoofed X-Forwarded-For must be ignored", rec.Code)
		}
	})

	t.Run("Spoofed X-Forwarded-For alone from trusted peer does not bypass IPFilter (rejected with 403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/contents", nil)
		// Even if direct connection is loopback, ExtractIPFromRealIPHeader only respects X-Real-IP, not XFF.
		req.RemoteAddr = "127.0.0.1:54321"
		req.Header.Set("X-Forwarded-For", "192.168.1.50")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: XFF header must not be used by ExtractIPFromRealIPHeader", rec.Code)
		}
	})
}

func TestRouter_RequestID(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("Propagates incoming X-Request-ID", func(t *testing.T) {
		customID := "test-req-id-12345"
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Request-ID", customID)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("X-Request-ID"); got != customID {
			t.Errorf("expected X-Request-ID %q, got %q", customID, got)
		}
	})

	t.Run("Generates X-Request-ID when not provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		got := rec.Header().Get("X-Request-ID")
		if got == "" {
			t.Errorf("expected non-empty X-Request-ID header")
		}
	})
}

func TestRouter_ErrorResponses(t *testing.T) {
	params, _, _, _, _, _ := setupTestRouter("")
	router := setupRouter(*params)

	t.Run("404 Not Found on unmapped route returns JSON error format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent/endpoint", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}

		var res map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to parse JSON error response: %v", err)
		}
		if _, ok := res["message"]; !ok {
			t.Errorf("expected error JSON to contain 'message' key, got %+v", res)
		}
	})

	t.Run("400 Bad Request on invalid parameter returns JSON error format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/invalid!id@here", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}

		var res map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to parse JSON error response: %v", err)
		}
		if _, ok := res["message"]; !ok {
			t.Errorf("expected error JSON to contain 'message' key, got %+v", res)
		}
	})
}
