package shared_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

func TestStripTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello world without tags",
			expected: "Hello world without tags",
		},
		{
			name:     "simple tags",
			input:    "<p>Hello <b>world</b>!</p>",
			expected: "Hello world!",
		},
		{
			name:     "nested tags with whitespace",
			input:    "<div>\n  <p>First paragraph</p>\n  <span>Second part</span>\n</div>",
			expected: "First paragraph Second part",
		},
		{
			name:     "attributes in tags",
			input:    `<a href="https://example.com" class="link">Click here</a>`,
			expected: "Click here",
		},
		{
			name:     "unclosed or malformed tags",
			input:    "text with <broken tag and closing > afterwards",
			expected: "text with afterwards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.StripTags(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "ascii string shorter than limit",
			input:    "hello",
			limit:    10,
			expected: "hello",
		},
		{
			name:     "ascii string longer than limit",
			input:    "hello world",
			limit:    5,
			expected: "hello",
		},
		{
			name:     "japanese multibyte string truncation",
			input:    "こんにちは世界！素晴らしい天気ですね",
			limit:    7,
			expected: "こんにちは世界",
		},
		{
			name:     "whitespace trimming before truncate",
			input:    "   spaced text   ",
			limit:    6,
			expected: "spaced",
		},
		{
			name:     "zero limit",
			input:    "some text",
			limit:    0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.Truncate(tt.input, tt.limit)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsPrimaryDomain(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"github repo url", "https://github.com/golang/go", true},
		{"openai domain", "https://openai.com/index/gpt-4o", true},
		{"anthropic subdomain", "https://news.anthropic.com/article", true},
		{"kubernetes", "https://kubernetes.io/docs", true},
		{"huggingface", "https://huggingface.co/models", true},
		{"invalid url", ":::not-a-valid-url:::", false},
		{"third party news blog", "https://techcrunch.com/2026/01/01/ai-news", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.IsPrimaryDomain(tt.url)
			if got != tt.expected {
				t.Errorf("IsPrimaryDomain(%q) = %v; want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestSetPrimaryDomainPatterns(t *testing.T) {
	// Restore default patterns at the end of the test
	origPatterns := []string{
		`(?i)github\.com/[^/]+/[^/]+`,
		`(?i)([a-z0-9-]+\.)?openai\.com`,
		`(?i)([a-z0-9-]+\.)?anthropic\.com`,
		`(?i)([a-z0-9-]+\.)?moonshot\.cn`,
		`(?i)kubernetes\.io`,
		`(?i)cncf\.io`,
		`(?i)aws\.amazon\.com`,
		`(?i)blog\.cloudflare\.com`,
		`(?i)qwenlm\.github\.io`,
		`(?i)([a-z0-9-]+\.)?nvidia\.com`,
		`(?i)([a-z0-9-]+\.)?amd\.com`,
		`(?i)([a-z0-9-]+\.)?intel\.com`,
		`(?i)huggingface\.co`,
		`(?i)arxiv\.org`,
		`(?i)([a-z0-9-]+\.)?google(blog)?\.(com|dev)`,
		`(?i)research\.google`,
		`(?i)([a-z0-9-]+\.)?microsoft\.com`,
	}
	defer shared.SetPrimaryDomainPatterns(origPatterns)

	// Custom patterns
	shared.SetPrimaryDomainPatterns([]string{`(?i)custom-primary\.example\.com`})

	if !shared.IsPrimaryDomain("https://custom-primary.example.com/announcement") {
		t.Errorf("expected custom pattern to match")
	}
	if shared.IsPrimaryDomain("https://other-domain.example.com") {
		t.Errorf("expected non-matching pattern to return false")
	}
}

func TestExtractPrimaryURL_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>No primary links here, only <a href="https://example.org">example</a></p></body></html>`))
	}))
	defer srv.Close()

	t.Setenv("BARE_WEB_PROXY_URL", "")

	client := shared.NewHTTPClient()
	_, err := shared.ExtractPrimaryURL(client, srv.URL)
	if err == nil {
		t.Fatal("expected error when no primary source URL is found, got nil")
	}
	if !strings.Contains(err.Error(), "no primary source URL found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFetchTextContent_HttpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("BARE_WEB_PROXY_URL", "")

	client := shared.NewHTTPClient()
	_, err := shared.FetchTextContent(client, srv.URL)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("unexpected error message: %v", err)
	}
}
