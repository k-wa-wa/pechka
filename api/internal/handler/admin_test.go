package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type mockPgContentRepo struct {
	createFn      func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
	updateFn      func(ctx context.Context, params pgRepo.UpdateContentParams) (*domain.Content, error)
	deleteFn      func(ctx context.Context, id string) error
	setArchivedFn func(ctx context.Context, id string, archived bool) (*domain.Content, error)
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
func (m *mockPgContentRepo) SetArchived(ctx context.Context, id string, archived bool) (*domain.Content, error) {
	return m.setArchivedFn(ctx, id, archived)
}

type mockReadContentRepo struct {
	listAdminFn func(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error)
}

func (m *mockReadContentRepo) ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, params)
	}
	return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
}

type mockPlaylistRefresher struct {
	refreshFn func(ctx context.Context) error
	called    int
}

func (m *mockPlaylistRefresher) Refresh(ctx context.Context) error {
	m.called++
	if m.refreshFn != nil {
		return m.refreshFn(ctx)
	}
	return nil
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
	return m.listTracksFn(ctx, contentID)
}
func (m *mockPgSubtitleRepo) GetTrack(ctx context.Context, trackID string) (*domain.SubtitleTrack, error) {
	return m.getTrackFn(ctx, trackID)
}
func (m *mockPgSubtitleRepo) SetTrackStatus(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error) {
	return m.setStatusFn(ctx, trackID, status)
}
func (m *mockPgSubtitleRepo) ListCuesByTrackID(ctx context.Context, trackID string) ([]*domain.SubtitleCue, error) {
	return m.listCuesFn(ctx, trackID)
}
func (m *mockPgSubtitleRepo) UpdateCue(ctx context.Context, params pgRepo.UpdateCueParams) (*domain.SubtitleCue, error) {
	return m.updateCueFn(ctx, params)
}
func (m *mockPgSubtitleRepo) InsertCue(ctx context.Context, params pgRepo.InsertCueParams) (*domain.SubtitleCue, error) {
	return m.insertCueFn(ctx, params)
}
func (m *mockPgSubtitleRepo) DeleteCue(ctx context.Context, id string) error {
	return m.deleteCueFn(ctx, id)
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
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
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

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"content_type":"video","title":"Test Video"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "bluray")
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

func TestAdminHandler_ListContents_DefaultLimitAndSort(t *testing.T) {
	readRepo := &mockReadContentRepo{
		listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
			if params.Limit != 20 {
				t.Errorf("expected default limit 20, got %d", params.Limit)
			}
			if params.Sort != domain.ContentSortUpdatedDesc {
				t.Errorf("expected default sort updated_desc, got %q", params.Sort)
			}
			return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithReadContentRepository(readRepo)
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

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
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
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
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

func TestAdminHandler_DeleteContent_Success(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		deleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	refresher := &mockPlaylistRefresher{}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(refresher)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/contents/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")

	if err := h.DeleteContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if refresher.called != 1 {
		t.Errorf("expected playlist refresher to be called once, got %d", refresher.called)
	}
}

func TestAdminHandler_DeleteContent_InternalError(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("db error")
		},
	}
	refresher := &mockPlaylistRefresher{}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(refresher)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/contents/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")

	err := h.DeleteContent(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
	if refresher.called != 0 {
		t.Errorf("playlist refresher should not be called on delete error")
	}
}

func TestAdminHandler_ListContents_EmptyReturnsArray(t *testing.T) {
	readRepo := &mockReadContentRepo{
		listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
			return &domain.AdminContentPage{
				Contents: nil,
				Total:    0,
			}, nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithReadContentRepository(readRepo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListContents(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Contents   []any   `json:"contents"`
		Total      int     `json:"total"`
		Limit      int     `json:"limit"`
		NextCursor *string `json:"next_cursor"`
		PrevCursor *string `json:"prev_cursor"`
		HasNext    bool    `json:"has_next"`
		HasPrev    bool    `json:"has_prev"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Contents == nil {
		t.Error("expected empty array, got null")
	}
}

func TestAdminHandler_ListContents_ReturnsPaginationMetadata(t *testing.T) {
	nextCur := "next-tok"
	prevCur := "prev-tok"
	readRepo := &mockReadContentRepo{
		listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
			if params.Limit != 10 {
				t.Errorf("expected limit 10, got %d", params.Limit)
			}
			if params.Sort != domain.ContentSortTitleAsc {
				t.Errorf("expected sort title_asc, got %q", params.Sort)
			}
			return &domain.AdminContentPage{
				Contents:   []*domain.Content{{ID: "1", ShortID: "s1"}},
				Total:      42,
				NextCursor: &nextCur,
				PrevCursor: &prevCur,
				HasNext:    true,
				HasPrev:    true,
			}, nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithReadContentRepository(readRepo)
	e := echo.New()
	// Legacy ?offset=20 during rolling deploy is ignored without error
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?limit=10&sort=title_asc&offset=20", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListContents(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Contents   []any   `json:"contents"`
		Total      int     `json:"total"`
		Limit      int     `json:"limit"`
		NextCursor *string `json:"next_cursor"`
		PrevCursor *string `json:"prev_cursor"`
		HasNext    bool    `json:"has_next"`
		HasPrev    bool    `json:"has_prev"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Total != 42 || result.Limit != 10 || !result.HasNext || !result.HasPrev {
		t.Errorf("unexpected response metadata: %+v", result)
	}
	if result.NextCursor == nil || *result.NextCursor != "next-tok" {
		t.Errorf("unexpected next_cursor: %v", result.NextCursor)
	}
	if result.PrevCursor == nil || *result.PrevCursor != "prev-tok" {
		t.Errorf("unexpected prev_cursor: %v", result.PrevCursor)
	}
}

func TestAdminHandler_ListContents_ReadRepoError(t *testing.T) {
	readRepo := &mockReadContentRepo{
		listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
			return nil, errors.New("db error")
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithReadContentRepository(readRepo)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListContents(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
}

func TestAdminHandler_ListContents_NilReadRepoReturns500(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListContents(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
}

func TestAdminHandler_ListContents_CursorAndSortValidation(t *testing.T) {
	validCursor, err := mongoRepo.EncodeAdminCursor(domain.ContentSortTitleAsc, &domain.Content{
		ShortID: "s1",
		Title:   "Alpha",
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	t.Run("all 6 valid sorts and after/before propagation", func(t *testing.T) {
		for _, sort := range []domain.ContentSort{
			domain.ContentSortUpdatedDesc,
			domain.ContentSortUpdatedAsc,
			domain.ContentSortCreatedDesc,
			domain.ContentSortCreatedAsc,
			domain.ContentSortTitleAsc,
			domain.ContentSortTitleDesc,
		} {
			var gotSort domain.ContentSort
			readRepo := &mockReadContentRepo{
				listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
					gotSort = params.Sort
					return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
				},
			}
			h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
				WithReadContentRepository(readRepo)
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?sort="+string(sort), nil)
			rec := httptest.NewRecorder()
			if err := h.ListContents(e.NewContext(req, rec)); err != nil {
				t.Fatalf("sort %s: %v", sort, err)
			}
			if gotSort != sort {
				t.Errorf("got sort %q, want %q", gotSort, sort)
			}
		}

		// after and before propagation
		var gotAfter, gotBefore *mongoRepo.AdminCursor
		readRepo := &mockReadContentRepo{
			listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				gotAfter = params.After
				gotBefore = params.Before
				return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
			},
		}
		h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
			WithReadContentRepository(readRepo)
		e := echo.New()

		reqAfter := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?sort=title_asc&after="+validCursor, nil)
		if err := h.ListContents(e.NewContext(reqAfter, httptest.NewRecorder())); err != nil {
			t.Fatalf("after failed: %v", err)
		}
		if gotAfter == nil || gotAfter.ID != "s1" || gotBefore != nil {
			t.Errorf("unexpected after/before: %+v / %+v", gotAfter, gotBefore)
		}

		reqBefore := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?sort=title_asc&before="+validCursor, nil)
		if err := h.ListContents(e.NewContext(reqBefore, httptest.NewRecorder())); err != nil {
			t.Fatalf("before failed: %v", err)
		}
		if gotBefore == nil || gotBefore.ID != "s1" || gotAfter != nil {
			t.Errorf("unexpected before/after: %+v / %+v", gotBefore, gotAfter)
		}
	})

	badURLs := []string{
		"/v1/admin/contents?sort=invalid_sort",
		"/v1/admin/contents?channel_id=BAD",
		"/v1/admin/contents?sort=title_asc&after=" + validCursor + "&before=" + validCursor,
		"/v1/admin/contents?sort=title_asc&after=not-valid-base64!",
		"/v1/admin/contents?sort=updated_desc&after=" + validCursor,
		"/v1/admin/contents?sort=updated_desc&before=" + validCursor,
	}
	for _, u := range badURLs {
		called := false
		readRepo := &mockReadContentRepo{
			listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				called = true
				return nil, nil
			},
		}
		h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
			WithReadContentRepository(readRepo)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, u, nil)
		err := h.ListContents(e.NewContext(req, httptest.NewRecorder()))
		if err == nil {
			t.Fatalf("expected 400 for %s, got nil", u)
		}
		if he := err.(*echo.HTTPError); he.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", u, he.Code)
		}
		if called {
			t.Errorf("%s: readRepo should not be called on 400", u)
		}
	}

	// Non-integer or out-of-range limit falls back to default 20
	for _, limitParam := range []string{"1.5", "0", "-5", "101", "abc", "10abc"} {
		var gotLimit int
		readRepo := &mockReadContentRepo{
			listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				gotLimit = params.Limit
				return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
			},
		}
		h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
			WithReadContentRepository(readRepo)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/v1/admin/contents?limit="+limitParam, nil)
		rec := httptest.NewRecorder()
		if err := h.ListContents(e.NewContext(req, rec)); err != nil {
			t.Fatalf("limit=%s: unexpected error: %v", limitParam, err)
		}
		if gotLimit != 20 {
			t.Errorf("limit=%s: got limit %d, want 20", limitParam, gotLimit)
		}
		var resp struct {
			Limit int `json:"limit"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Limit != 20 {
			t.Errorf("limit=%s: response limit=%d, want 20", limitParam, resp.Limit)
		}
	}
}

func TestAdminHandler_UpdateSubtitleTrackStatus_InvalidStatus(t *testing.T) {
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	e := echo.New()

	body := `{"status":"bogus"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/subtitles/1/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("track_id")
	c.SetParamValues("1")

	err := h.UpdateSubtitleTrackStatus(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestAdminHandler_UpdateSubtitleTrackStatus_Published(t *testing.T) {
	var gotStatus domain.SubtitleTrackStatus
	subtitleRepo := &mockPgSubtitleRepo{
		setStatusFn: func(_ context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error) {
			gotStatus = status
			return &domain.SubtitleTrack{ID: trackID, Status: status}, nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, subtitleRepo, newSnowflakeNode(t))
	e := echo.New()

	body := `{"status":"published"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/subtitles/1/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("track_id")
	c.SetParamValues("1")

	if err := h.UpdateSubtitleTrackStatus(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotStatus != domain.SubtitleTrackStatusPublished {
		t.Errorf("expected status published, got %q", gotStatus)
	}
}

func TestAdminHandler_ArchiveContent_Success(t *testing.T) {
	var gotArchived bool
	contentRepo := &mockPgContentRepo{
		setArchivedFn: func(_ context.Context, id string, archived bool) (*domain.Content, error) {
			gotArchived = archived
			return &domain.Content{ID: id, Tags: []string{}}, nil
		},
	}
	refresher := &mockPlaylistRefresher{}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(refresher)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents/123/archive", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")

	if err := h.ArchiveContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !gotArchived {
		t.Error("expected archived=true to be passed to repo")
	}
	if refresher.called != 1 {
		t.Errorf("expected playlist refresher to be called once, got %d", refresher.called)
	}
	if !strings.Contains(rec.Body.String(), `"tags":[]`) {
		t.Errorf("expected archive response JSON to contain \"tags\":[], got %s", rec.Body.String())
	}
}

func TestAdminHandler_ArchiveContent_NotFound(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		setArchivedFn: func(_ context.Context, _ string, _ bool) (*domain.Content, error) {
			return nil, pgx.ErrNoRows
		},
	}
	refresher := &mockPlaylistRefresher{}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(refresher)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents/999/archive", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := h.ArchiveContent(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", he.Code)
	}
	if refresher.called != 0 {
		t.Errorf("playlist refresher should not be called on 404")
	}
}

func TestAdminHandler_UnarchiveContent_Success(t *testing.T) {
	var gotArchived bool
	contentRepo := &mockPgContentRepo{
		setArchivedFn: func(_ context.Context, id string, archived bool) (*domain.Content, error) {
			gotArchived = archived
			return &domain.Content{ID: id}, nil
		},
	}
	refresher := &mockPlaylistRefresher{}

	h := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(refresher)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents/123/unarchive", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")

	if err := h.UnarchiveContent(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotArchived {
		t.Error("expected archived=false to be passed to repo")
	}
	if refresher.called != 1 {
		t.Errorf("expected playlist refresher to be called once, got %d", refresher.called)
	}
}

func TestAdminHandler_PlaylistRefresher_NilAndErrorReturn500(t *testing.T) {
	contentRepo := &mockPgContentRepo{
		deleteFn: func(_ context.Context, _ string) error { return nil },
		setArchivedFn: func(_ context.Context, id string, _ bool) (*domain.Content, error) {
			return &domain.Content{ID: id}, nil
		},
	}
	e := echo.New()

	// nil refresher -> 500
	hNil := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t))
	for _, tc := range []struct {
		method string
		path   string
		fn     func(echo.Context) error
	}{
		{http.MethodDelete, "/v1/admin/contents/123", hNil.DeleteContent},
		{http.MethodPost, "/v1/admin/contents/123/archive", hNil.ArchiveContent},
		{http.MethodPost, "/v1/admin/contents/123/unarchive", hNil.UnarchiveContent},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		c := e.NewContext(req, httptest.NewRecorder())
		c.SetParamNames("id")
		c.SetParamValues("123")
		err := tc.fn(c)
		if err == nil || err.(*echo.HTTPError).Code != http.StatusInternalServerError {
			t.Errorf("%s %s with nil refresher: expected 500, got %v", tc.method, tc.path, err)
		}
	}

	// failing refresher -> 500
	failingRefresher := &mockPlaylistRefresher{
		refreshFn: func(_ context.Context) error { return errors.New("minio error") },
	}
	hFail := handler.NewAdminHandler(contentRepo, &mockPgDiscRepo{}, &mockPgSubtitleRepo{}, newSnowflakeNode(t)).
		WithPlaylistRefresher(failingRefresher)
	for _, tc := range []struct {
		method string
		path   string
		fn     func(echo.Context) error
	}{
		{http.MethodDelete, "/v1/admin/contents/123", hFail.DeleteContent},
		{http.MethodPost, "/v1/admin/contents/123/archive", hFail.ArchiveContent},
		{http.MethodPost, "/v1/admin/contents/123/unarchive", hFail.UnarchiveContent},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		c := e.NewContext(req, httptest.NewRecorder())
		c.SetParamNames("id")
		c.SetParamValues("123")
		err := tc.fn(c)
		if err == nil || err.(*echo.HTTPError).Code != http.StatusInternalServerError {
			t.Errorf("%s %s with failing refresher: expected 500, got %v", tc.method, tc.path, err)
		}
	}
}

func TestAdminHandler_SubtitleCueNotFoundReturns404(t *testing.T) {
	subRepo := &mockPgSubtitleRepo{
		insertCueFn: func(_ context.Context, _ pgRepo.InsertCueParams) (*domain.SubtitleCue, error) {
			return nil, pgx.ErrNoRows
		},
		deleteCueFn: func(_ context.Context, _ string) error {
			return pgx.ErrNoRows
		},
	}
	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, subRepo, newSnowflakeNode(t))
	e := echo.New()

	reqInsert := httptest.NewRequest(http.MethodPost, "/v1/admin/subtitles/t1/cues", strings.NewReader(`{"seq":1,"start_ms":0,"end_ms":1000,"text":"hi"}`))
	reqInsert.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	cInsert := e.NewContext(reqInsert, httptest.NewRecorder())
	cInsert.SetParamNames("track_id")
	cInsert.SetParamValues("t1")
	if err := h.InsertSubtitleCue(cInsert); err == nil || err.(*echo.HTTPError).Code != http.StatusNotFound {
		t.Errorf("InsertSubtitleCue expected 404, got %v", err)
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "/v1/admin/subtitles/cues/c1", nil)
	cDel := e.NewContext(reqDel, httptest.NewRecorder())
	cDel.SetParamNames("cue_id")
	cDel.SetParamValues("c1")
	if err := h.DeleteSubtitleCue(cDel); err == nil || err.(*echo.HTTPError).Code != http.StatusNotFound {
		t.Errorf("DeleteSubtitleCue expected 404, got %v", err)
	}
}

func TestAdminHandler_ListSubtitleCues_EmptyReturnsArray(t *testing.T) {
	subtitleRepo := &mockPgSubtitleRepo{
		listCuesFn: func(_ context.Context, _ string) ([]*domain.SubtitleCue, error) {
			return nil, nil
		},
	}

	h := handler.NewAdminHandler(&mockPgContentRepo{}, &mockPgDiscRepo{}, subtitleRepo, newSnowflakeNode(t))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/subtitles/1/cues", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("track_id")
	c.SetParamValues("1")

	if err := h.ListSubtitleCues(c); err != nil {
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
