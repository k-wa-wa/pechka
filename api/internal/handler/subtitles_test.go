package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
)

type mockMongoSubtitleRepo struct {
	getFn func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

func (m *mockMongoSubtitleRepo) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	return m.getFn(ctx, shortID, language)
}

func TestSubtitlesHandler_GetVTT_NotFound(t *testing.T) {
	mock := &mockMongoSubtitleRepo{
		getFn: func(_ context.Context, _, _ string) (*domain.MongoSubtitle, error) {
			return nil, mongo.ErrNoDocuments
		},
	}

	h := handler.NewSubtitlesHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents/abc/subtitles/ja", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("short_id", "lang")
	c.SetParamValues("abc", "ja")

	err := h.GetVTT(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", he.Code)
	}
}

func TestSubtitlesHandler_GetVTT_RendersCues(t *testing.T) {
	mock := &mockMongoSubtitleRepo{
		getFn: func(_ context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return &domain.MongoSubtitle{
				ShortID:  shortID,
				Language: language,
				Status:   "published",
				Cues: []domain.MongoCue{
					{StartMs: 0, EndMs: 1500, Text: "こんにちは"},
					{StartMs: 1500, EndMs: 3000, Text: "世界"},
				},
			}, nil
		},
	}

	h := handler.NewSubtitlesHandler(mock)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/contents/abc/subtitles/ja", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("short_id", "lang")
	c.SetParamValues("abc", "ja")

	if err := h.GetVTT(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, "text/vtt") {
		t.Errorf("expected text/vtt content type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "WEBVTT\n\n") {
		t.Errorf("expected body to start with WEBVTT header, got %q", body)
	}
	if !strings.Contains(body, "00:00:00.000 --> 00:00:01.500\nこんにちは") {
		t.Errorf("expected first cue to be rendered, got %q", body)
	}
}
