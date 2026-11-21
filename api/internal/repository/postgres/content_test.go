package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestCreate_MissingChannelIDReturnsError(t *testing.T) {
	repo := NewContentRepository(nil)
	_, err := repo.Create(context.Background(), CreateContentParams{
		ShortID:     "short-1",
		ContentType: domain.ContentTypeVideo,
		ChannelID:   "   ",
		Title:       "Test",
		Status:      domain.ContentStatusPending,
	})
	if err == nil {
		t.Fatal("expected error when ChannelID is empty, got nil")
	}
	if !strings.Contains(err.Error(), "channel_id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildListContentsWhere(t *testing.T) {
	statusReady := domain.ContentStatusReady
	channelTechFeed := "tech-feed"

	where, args, nextIdx := buildListContentsWhere(ListContentsParams{
		Status:          &statusReady,
		ChannelID:       &channelTechFeed,
		IncludeArchived: false,
	})

	wantWhere := " WHERE deleted_at IS NULL AND status = $1 AND channel_id = $2 AND archived_at IS NULL"
	if where != wantWhere {
		t.Errorf("got where %q, want %q", where, wantWhere)
	}
	if len(args) != 2 || args[0] != statusReady || args[1] != channelTechFeed {
		t.Errorf("unexpected args: %v", args)
	}
	if nextIdx != 3 {
		t.Errorf("got nextIdx %d, want 3", nextIdx)
	}

	whereAdmin, argsAdmin, nextIdxAdmin := buildListContentsWhere(ListContentsParams{
		IncludeArchived: true,
	})
	if whereAdmin != " WHERE deleted_at IS NULL" {
		t.Errorf("expected deleted_at IS NULL even with IncludeArchived=true, got %q", whereAdmin)
	}
	if len(argsAdmin) != 0 || nextIdxAdmin != 1 {
		t.Errorf("unexpected args/nextIdx: %v / %d", argsAdmin, nextIdxAdmin)
	}
}

func TestContentQueriesExcludeDeletedAt(t *testing.T) {
	title := "Updated"
	qUpdate, _, ok := buildUpdateContentQuery(UpdateContentParams{ID: "id-1", Title: &title})
	if !ok || !strings.Contains(qUpdate, "deleted_at IS NULL") {
		t.Errorf("Update query must require deleted_at IS NULL, got %q", qUpdate)
	}
	if !strings.Contains(deleteContentQuery, "UPDATE contents SET deleted_at = CURRENT_TIMESTAMP") ||
		!strings.Contains(deleteContentQuery, "deleted_at IS NULL") {
		t.Errorf("Delete query must be soft-delete, got %q", deleteContentQuery)
	}
	if !strings.Contains(setArchivedTrueQuery, "deleted_at IS NULL") ||
		!strings.Contains(setArchivedFalseQuery, "deleted_at IS NULL") {
		t.Errorf("SetArchived queries must require deleted_at IS NULL")
	}
	if !strings.Contains(getContentByIDQuery, "deleted_at IS NULL") {
		t.Errorf("GetByID query must require deleted_at IS NULL, got %q", getContentByIDQuery)
	}
	if !strings.Contains(LatestPlaylistQuery, "c.status = 'ready'") ||
		!strings.Contains(LatestPlaylistQuery, "c.archived_at IS NULL") ||
		!strings.Contains(LatestPlaylistQuery, "c.deleted_at IS NULL") ||
		!strings.Contains(LatestPlaylistQuery, "video_variants v") ||
		!strings.Contains(LatestPlaylistQuery, "v.variant_type = 'master'") ||
		!strings.Contains(LatestPlaylistQuery, "ORDER BY c.published_at DESC NULLS LAST, c.updated_at DESC, c.short_id ASC") {
		t.Errorf("LatestPlaylistQuery unexpected: %q", LatestPlaylistQuery)
	}
	if strings.Index(LatestPlaylistQuery, "v.variant_type = 'master'") > strings.Index(LatestPlaylistQuery, "LIMIT $1") {
		t.Errorf("LatestPlaylistQuery must filter master variant before LIMIT: %q", LatestPlaylistQuery)
	}
}

type fakeScanner struct {
	vals []any
}

func (s fakeScanner) Scan(dest ...any) error {
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = s.vals[i].(string)
		case **time.Time:
			if s.vals[i] == nil {
				*d = nil
			} else {
				v := s.vals[i].(*time.Time)
				*d = v
			}
		}
	}
	return nil
}

func TestScanPlaylistItem_AllowsNullPublishedAt(t *testing.T) {
	now := time.Now().UTC()
	itemWithTime, err := scanPlaylistItem(fakeScanner{vals: []any{"s1", "Title 1", &now}})
	if err != nil {
		t.Fatalf("unexpected error with non-null published_at: %v", err)
	}
	if itemWithTime.PublishedAt == nil || !itemWithTime.PublishedAt.Equal(now) {
		t.Errorf("unexpected publishedAt: %v", itemWithTime.PublishedAt)
	}

	itemWithNull, err := scanPlaylistItem(fakeScanner{vals: []any{"s2", "Title 2", nil}})
	if err != nil {
		t.Fatalf("unexpected error with null published_at: %v", err)
	}
	if itemWithNull.ShortID != "s2" || itemWithNull.Title != "Title 2" || itemWithNull.PublishedAt != nil {
		t.Errorf("unexpected itemWithNull: %+v", itemWithNull)
	}
}

type mockPlaylistRows struct {
	items []PlaylistItem
	idx   int
}

func (r *mockPlaylistRows) Next() bool {
	if r.idx < len(r.items) {
		r.idx++
		return true
	}
	return false
}

func (r *mockPlaylistRows) Scan(dest ...any) error {
	item := r.items[r.idx-1]
	*(dest[0].(*string)) = item.ShortID
	*(dest[1].(*string)) = item.Title
	*(dest[2].(**time.Time)) = item.PublishedAt
	return nil
}

func (r *mockPlaylistRows) Err() error { return nil }
func (r *mockPlaylistRows) Close()     {}

type mockPlaylistTx struct {
	events *[]string
	items  []PlaylistItem
}

func (tx *mockPlaylistTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "pg_advisory_xact_lock") {
		if len(args) != 1 || args[0] != LatestPlaylistLockKey {
			return pgconn.CommandTag{}, errors.New("unexpected lock key")
		}
		*tx.events = append(*tx.events, "lock")
	}
	return pgconn.CommandTag{}, nil
}

func (tx *mockPlaylistTx) Query(_ context.Context, sql string, args ...any) (playlistRows, error) {
	if sql != LatestPlaylistQuery {
		return nil, errors.New("unexpected query")
	}
	if len(args) != 1 || args[0] != LatestPlaylistLimit {
		return nil, errors.New("unexpected limit")
	}
	*tx.events = append(*tx.events, "query")
	return &mockPlaylistRows{items: tx.items}, nil
}

func (tx *mockPlaylistTx) Commit(_ context.Context) error {
	*tx.events = append(*tx.events, "commit")
	return nil
}

func (tx *mockPlaylistTx) Rollback(_ context.Context) error {
	*tx.events = append(*tx.events, "rollback")
	return nil
}

type mockPlaylistBeginner struct {
	tx      *mockPlaylistTx
	gotOpts pgx.TxOptions
	events  *[]string
}

func (b *mockPlaylistBeginner) BeginTx(_ context.Context, txOptions pgx.TxOptions) (playlistTx, error) {
	b.gotOpts = txOptions
	*b.events = append(*b.events, "begin")
	return b.tx, nil
}

func TestRefreshLatestPlaylistWithBeginner_OrderAndNullPublishedAt(t *testing.T) {
	events := []string{}
	now := time.Now().UTC()
	beginner := &mockPlaylistBeginner{
		events: &events,
		tx: &mockPlaylistTx{
			events: &events,
			items: []PlaylistItem{
				{ShortID: "s1", Title: "Published Video", PublishedAt: &now},
				{ShortID: "s2", Title: "Ready Without PublishedAt", PublishedAt: nil},
			},
		},
	}

	var uploaded string
	err := refreshLatestPlaylistWithBeginner(context.Background(), beginner, func(_ context.Context, playlist string) error {
		events = append(events, "upload")
		uploaded = playlist
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if beginner.gotOpts.IsoLevel != pgx.ReadCommitted {
		t.Errorf("expected ReadCommitted isolation level, got %v", beginner.gotOpts.IsoLevel)
	}
	wantOrder := "begin,lock,query,upload,commit,rollback"
	if got := strings.Join(events, ","); got != wantOrder {
		t.Errorf("execution order = %q, want %q", got, wantOrder)
	}
	wantPlaylist := "#EXTM3U\n#EXTINF:-1,Published Video\ns1/master.m3u8\n#EXTINF:-1,Ready Without PublishedAt\ns2/master.m3u8\n"
	if uploaded != wantPlaylist {
		t.Errorf("uploaded playlist = %q, want %q", uploaded, wantPlaylist)
	}
}

func TestBuildLatestPlaylist_RelativeURIResolutionAndSanitization(t *testing.T) {
	pl := BuildLatestPlaylist([]PlaylistItem{
		{ShortID: "abc123", Title: "Normal Title"},
		{ShortID: "inj1", Title: "Line 1\n#EXT-X-KEY:METHOD=AES-128\r\nLine 2\rLine 3"},
	})
	want := "#EXTM3U\n#EXTINF:-1,Normal Title\nabc123/master.m3u8\n#EXTINF:-1,Line 1 #EXT-X-KEY:METHOD=AES-128 Line 2 Line 3\ninj1/master.m3u8\n"
	if pl != want {
		t.Fatalf("got playlist:\n%q\nwant:\n%q", pl, want)
	}

	baseURL, err := url.Parse("https://pechka.example/resources/hls/latest.m3u8")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	var resolved []string
	for _, line := range strings.Split(strings.TrimSpace(pl), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ref, err := url.Parse(line)
		if err != nil {
			t.Fatalf("parse entry %q: %v", line, err)
		}
		resolved = append(resolved, baseURL.ResolveReference(ref).Path)
	}
	wantPaths := []string{
		"/resources/hls/abc123/master.m3u8",
		"/resources/hls/inj1/master.m3u8",
	}
	if strings.Join(resolved, ",") != strings.Join(wantPaths, ",") {
		t.Errorf("resolved paths = %v, want %v", resolved, wantPaths)
	}
}

type fullContentRowScanner struct {
	vals []any
}

func (s fullContentRowScanner) Scan(dest ...any) error {
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = s.vals[i].(string)
		case **string:
			if s.vals[i] == nil {
				*d = nil
			} else {
				v := s.vals[i].(*string)
				*d = v
			}
		case *domain.ContentType:
			*d = s.vals[i].(domain.ContentType)
		case *domain.ContentStatus:
			*d = s.vals[i].(domain.ContentStatus)
		case **int:
			if s.vals[i] == nil {
				*d = nil
			} else {
				v := s.vals[i].(*int)
				*d = v
			}
		case *[]string:
			if s.vals[i] == nil {
				*d = nil
			} else {
				*d = s.vals[i].([]string)
			}
		case **time.Time:
			if s.vals[i] == nil {
				*d = nil
			} else {
				v := s.vals[i].(*time.Time)
				*d = v
			}
		case *time.Time:
			*d = s.vals[i].(time.Time)
		default:
			if ns, ok := d.(interface{ Scan(any) error }); ok {
				if err := ns.Scan(s.vals[i]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func TestScanContent_NullNullableFieldsNormalized(t *testing.T) {
	for _, col := range []string{"COALESCE(title, '')", "COALESCE(description, '')", "COALESCE(is_360, false)", "COALESCE(tags, '{}')"} {
		if !strings.Contains(contentColumns, col) {
			t.Errorf("contentColumns missing %q: %s", col, contentColumns)
		}
	}

	now := time.Now().UTC()
	c, err := scanContent(fullContentRowScanner{
		vals: []any{
			"uuid-1",
			"short-1",
			domain.ContentTypeVideo,
			"bluray",
			nil, // disc_id
			nil, // title NULL
			nil, // description NULL
			nil, // duration_seconds
			nil, // is_360 NULL
			nil, // tags NULL
			domain.ContentStatusReady,
			nil,  // document_key
			nil,  // document_mime_type
			nil,  // published_at
			&now, // archived_at
			now,
			now,
		},
	})
	if err != nil {
		t.Fatalf("scanContent failed on nullable columns: %v", err)
	}
	if c.Title != "" || c.Description != "" || c.Is360 {
		t.Errorf("unexpected normalized scalar fields: %+v", c)
	}
	if c.Tags == nil || len(c.Tags) != 0 {
		t.Fatalf("expected non-nil empty Tags slice, got %#v", c.Tags)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(raw), `"tags":[]`) {
		t.Errorf("expected serialized JSON to contain \"tags\":[], got %s", string(raw))
	}
}

func TestRefreshLatestPlaylistWithBeginner_UploadErrorRollsBack(t *testing.T) {
	events := []string{}
	beginner := &mockPlaylistBeginner{
		events: &events,
		tx: &mockPlaylistTx{
			events: &events,
			items:  []PlaylistItem{{ShortID: "s1", Title: "Video"}},
		},
	}

	err := refreshLatestPlaylistWithBeginner(context.Background(), beginner, func(_ context.Context, _ string) error {
		events = append(events, "upload")
		return errors.New("minio down")
	})
	if err == nil || !strings.Contains(err.Error(), "minio down") {
		t.Fatalf("expected upload error, got %v", err)
	}
	wantOrder := "begin,lock,query,upload,rollback"
	if got := strings.Join(events, ","); got != wantOrder {
		t.Errorf("execution order = %q, want %q", got, wantOrder)
	}
}

func TestRefreshLatestPlaylistWithTimeout_DeadlineReleasesLock(t *testing.T) {
	if LatestPlaylistTimeout <= 0 {
		t.Fatalf("LatestPlaylistTimeout must be positive, got %v", LatestPlaylistTimeout)
	}
	events := []string{}
	beginner := &mockPlaylistBeginner{
		events: &events,
		tx: &mockPlaylistTx{
			events: &events,
			items:  []PlaylistItem{{ShortID: "s1", Title: "Video"}},
		},
	}

	var hasDeadline bool
	err := refreshLatestPlaylistWithTimeout(context.Background(), beginner, 20*time.Millisecond, func(ctx context.Context, _ string) error {
		_, hasDeadline = ctx.Deadline()
		events = append(events, "upload_wait")
		<-ctx.Done()
		return ctx.Err()
	})
	if !hasDeadline {
		t.Fatal("expected uploadFn context to have a deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if got := strings.Join(events, ","); got != "begin,lock,query,upload_wait,rollback" {
		t.Errorf("expected rollback without commit on deadline, got %q", got)
	}
}
