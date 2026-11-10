package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

func TestCollector_FromRSS(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Sample Feed</title>
    <link>https://example.com</link>
    <description>Test RSS Feed</description>
    <item>
      <title>Go 1.26 Released</title>
      <link>https://example.com/posts/go-126</link>
      <description>&lt;p&gt;Go 1.26 brings great new features.&lt;/p&gt;</description>
      <pubDate>` + time.Now().UTC().Format(time.RFC1123Z) + `</pubDate>
    </item>
    <item>
      <title>Old Post</title>
      <link>https://example.com/posts/old</link>
      <description>Too old post</description>
      <pubDate>` + time.Now().UTC().AddDate(0, 0, -10).Format(time.RFC1123Z) + `</pubDate>
    </item>
  </channel>
</rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer srv.Close()

	c := &collector{
		client: shared.NewHTTPClient(),
		since:  time.Now().UTC().AddDate(0, 0, -2),
	}

	candidates, err := c.fromRSS("Sample Publisher", srv.URL, true)
	if err != nil {
		t.Fatalf("fromRSS failed: %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate within range, got %d", len(candidates))
	}

	cand := candidates[0]
	if cand.Title != "Go 1.26 Released" {
		t.Errorf("unexpected title: %q", cand.Title)
	}
	if cand.URL != "https://example.com/posts/go-126" {
		t.Errorf("unexpected URL: %q", cand.URL)
	}
	if cand.Publisher != "Sample Publisher" {
		t.Errorf("unexpected Publisher: %q", cand.Publisher)
	}
	if !cand.IsPrimary {
		t.Errorf("expected is_primary true")
	}
	if !strings.Contains(cand.Summary, "Go 1.26 brings great new features.") {
		t.Errorf("unexpected Summary: %q", cand.Summary)
	}
}
