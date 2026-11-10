package shared_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

func TestShortIDFor(t *testing.T) {
	t.Run("standard key", func(t *testing.T) {
		sourceKey := "tech-feed:2026-08-02"

		id1 := shared.ShortIDFor(sourceKey)
		id2 := shared.ShortIDFor(sourceKey)

		if !strings.HasPrefix(id1, "tech-feed-2026-08-02-") {
			t.Errorf("expected prefix tech-feed-2026-08-02-, got %s", id1)
		}

		if id1 == id2 {
			t.Errorf("expected random suffix to be different, got identical: %s", id1)
		}

		if len(id1) > shared.ShortIDMax {
			t.Errorf("expected length <= %d, got %d", shared.ShortIDMax, len(id1))
		}
	})

	t.Run("empty source key defaults to tech-feed", func(t *testing.T) {
		id := shared.ShortIDFor("")
		if !strings.HasPrefix(id, "tech-feed-") {
			t.Errorf("expected prefix tech-feed-, got %s", id)
		}
	})

	t.Run("very long source key truncated with hash", func(t *testing.T) {
		longKey := strings.Repeat("very-long-source-key-title-with-many-words-and-symbols-", 5)
		id := shared.ShortIDFor(longKey)
		if len(id) > shared.ShortIDMax {
			t.Errorf("expected length <= %d, got %d (%s)", shared.ShortIDMax, len(id), id)
		}
	})
}

func TestStorageKeys(t *testing.T) {
	shortID := "tech-feed-2026-11-10-abc123"

	prefix := shared.HLSPrefix(shortID)
	expectedPrefix := "resources/hls/tech-feed-2026-11-10-abc123"
	if prefix != expectedPrefix {
		t.Errorf("expected %q, got %q", expectedPrefix, prefix)
	}

	thumbKey := shared.ThumbnailKey(shortID)
	expectedThumbKey := "thumbnails/tech-feed-2026-11-10-abc123/thumb_01.jpg"
	if thumbKey != expectedThumbKey {
		t.Errorf("expected %q, got %q", expectedThumbKey, thumbKey)
	}
}

func TestMasterPlaylist(t *testing.T) {
	variants := []shared.Variant{
		{Name: "original", Bandwidth: 0, Resolution: "", Codecs: ""},
		{Name: "720p", Bandwidth: 3128000, Resolution: "1280x720", Codecs: "avc1.4d001f,mp4a.40.2"},
	}

	pl := shared.MasterPlaylist(variants)
	if !strings.HasPrefix(pl, "#EXTM3U\n#EXT-X-VERSION:3\n") {
		t.Errorf("missing standard header: %s", pl)
	}
	if !strings.Contains(pl, "#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS=\"avc1.640028,mp4a.40.2\"\noriginal.m3u8\n") {
		t.Errorf("missing original variant stream inf: %s", pl)
	}
	if !strings.Contains(pl, "#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS=\"avc1.4d001f,mp4a.40.2\"\n720p.m3u8\n") {
		t.Errorf("missing 720p variant stream inf: %s", pl)
	}
}

func TestNormalizeForTTS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replace technical terms",
			input:    "GitHub と AWS の API を使う",
			expected: "ギットハブ と エーダブリューエス の エイピーアイ を使う",
		},
		{
			name:     "version replacement",
			input:    "Kubernetes v1.30 released",
			expected: "クーバネティス バージョン 1.30 released",
		},
		{
			name:     "empty text",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.NormalizeForTTS(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeForTTS(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadScript(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid script JSON", func(t *testing.T) {
		validJSON := `{
			"digest_date": "2026-11-10",
			"title": "Sample Script",
			"sections": [
				{
					"seq": 1,
					"slide": {"layout": "title", "title": "Welcome"},
					"narration": [{"text": "Hello world"}]
				}
			]
		}`
		filePath := filepath.Join(tmpDir, "valid_script.json")
		if err := os.WriteFile(filePath, []byte(validJSON), 0o644); err != nil {
			t.Fatal(err)
		}

		s, err := shared.LoadScript(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Title != "Sample Script" || len(s.Sections) != 1 {
			t.Errorf("unexpected script contents: %+v", s)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := shared.LoadScript(filepath.Join(tmpDir, "non_existent.json"))
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid json syntax", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "bad.json")
		if err := os.WriteFile(invalidPath, []byte("{invalid-json"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := shared.LoadScript(invalidPath)
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})
}
