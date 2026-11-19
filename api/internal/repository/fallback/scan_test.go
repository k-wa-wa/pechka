package fallback

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type mockRowScanner struct {
	values []any
	err    error
}

func (m *mockRowScanner) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	if len(dest) != len(m.values) {
		return fmt.Errorf("Scan dest count %d != values count %d", len(dest), len(m.values))
	}
	for i, d := range dest {
		val := m.values[i]
		if val == nil {
			continue
		}
		switch target := d.(type) {
		case *string:
			*target = val.(string)
		case **string:
			s := val.(string)
			*target = &s
		case *int:
			*target = val.(int)
		case **int:
			n := val.(int)
			*target = &n
		case *bool:
			*target = val.(bool)
		case *[]string:
			*target = val.([]string)
		case *[]byte:
			*target = val.([]byte)
		case *time.Time:
			*target = val.(time.Time)
		case **time.Time:
			t := val.(time.Time)
			*target = &t
		case *domain.ContentType:
			*target = domain.ContentType(val.(string))
		case *domain.ContentStatus:
			*target = domain.ContentStatus(val.(string))
		default:
			return fmt.Errorf("unsupported dest type at index %d: %T", i, d)
		}
	}
	return nil
}

func TestScanMongoContent_AllColumns(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	pub := now.Add(-1 * time.Hour)
	dur := 120
	disc := "Disc 1"
	thumb := "thumbnails/c1.jpg"
	docKey := "documents/c1.pdf"
	docMime := "application/pdf"

	variantsJSON, err := json.Marshal([]domain.MongoVariant{
		{
			VariantType: "hls",
			HLSKey:      "videos/c1/master.m3u8",
			Bandwidth:   intPtr(2500000),
			Resolution:  strPtr("1920x1080"),
			Codecs:      strPtr("avc1.640028,mp4a.40.2"),
		},
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// 18 columns in baseContentSelect order:
	// c.short_id, c.content_type, c.channel_id, channel_name, c.title, c.description,
	// c.duration_seconds, c.is_360, c.tags, c.status, disc_label, variants_json,
	// c.thumbnail_key, c.document_key, c.document_mime_type, c.published_at, c.updated_at, has_subtitles
	scanner := &mockRowScanner{
		values: []any{
			"c1_short",
			"video",
			"ch_1",
			"Channel One",
			"Test Title",
			"Test Description",
			dur,
			true,
			[]string{"tag1", "tag2"},
			"ready",
			disc,
			variantsJSON,
			thumb,
			docKey,
			docMime,
			pub,
			now,
			true,
		},
	}

	content, err := scanMongoContent(scanner)
	if err != nil {
		t.Fatalf("scanMongoContent failed: %v", err)
	}

	if content.ID != "c1_short" {
		t.Errorf("expected ID c1_short, got %s", content.ID)
	}
	if content.ContentType != domain.ContentTypeVideo {
		t.Errorf("expected video, got %s", content.ContentType)
	}
	if content.ChannelName != "Channel One" {
		t.Errorf("expected Channel One, got %s", content.ChannelName)
	}
	if content.Title != "Test Title" || content.Description != "Test Description" {
		t.Errorf("unexpected title/desc: %s / %s", content.Title, content.Description)
	}
	if content.DurationSeconds == nil || *content.DurationSeconds != 120 {
		t.Errorf("unexpected duration: %v", content.DurationSeconds)
	}
	if !content.Is360 {
		t.Errorf("expected is_360 true")
	}
	if len(content.Tags) != 2 || content.Tags[0] != "tag1" {
		t.Errorf("unexpected tags: %v", content.Tags)
	}
	if content.Status != domain.ContentStatusReady {
		t.Errorf("expected status ready, got %s", content.Status)
	}
	if content.DiscLabel == nil || *content.DiscLabel != "Disc 1" {
		t.Errorf("unexpected disc label: %v", content.DiscLabel)
	}
	if len(content.Variants) != 1 || content.Variants[0].VariantType != "hls" {
		t.Errorf("unexpected variants: %v", content.Variants)
	}
	if content.ThumbnailKey == nil || *content.ThumbnailKey != thumb {
		t.Errorf("unexpected thumbnail: %v", content.ThumbnailKey)
	}
	if !content.HasSubtitles {
		t.Errorf("expected has_subtitles true")
	}
}

func TestScanMongoContent_NullableColumns(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	scanner := &mockRowScanner{
		values: []any{
			"c2_short",
			"document",
			"ch_2",
			"Channel Two",
			"Doc Title",
			"",
			nil, // duration_seconds NULL
			false,
			nil, // tags NULL -> empty slice
			"pending",
			nil, // disc_label NULL
			[]byte("[]"),
			nil, // thumbnail_key NULL
			nil, // document_key NULL
			nil, // document_mime_type NULL
			nil, // published_at NULL
			now,
			false, // has_subtitles
		},
	}

	content, err := scanMongoContent(scanner)
	if err != nil {
		t.Fatalf("scanMongoContent failed: %v", err)
	}

	if content.DurationSeconds != nil {
		t.Errorf("expected nil duration, got %v", content.DurationSeconds)
	}
	if content.DiscLabel != nil {
		t.Errorf("expected nil disc_label, got %v", content.DiscLabel)
	}
	if content.ThumbnailKey != nil {
		t.Errorf("expected nil thumbnail_key, got %v", content.ThumbnailKey)
	}
	if content.PublishedAt != nil {
		t.Errorf("expected nil published_at, got %v", content.PublishedAt)
	}
	if content.Tags == nil || len(content.Tags) != 0 {
		t.Errorf("expected empty tags slice, got %v", content.Tags)
	}
	if content.Variants == nil || len(content.Variants) != 0 {
		t.Errorf("expected empty variants slice, got %v", content.Variants)
	}
}

func TestScanMongoContent_ScanError(t *testing.T) {
	scanner := &mockRowScanner{
		err: errors.New("db error"),
	}
	_, err := scanMongoContent(scanner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }
