package shared_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

func TestFetchTextContent_UsesProxyEndpoint(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("url")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><p>hello world primary content here</p></body></html>"))
	}))
	defer srv.Close()

	t.Setenv("BARE_WEB_PROXY_URL", srv.URL)

	client := shared.NewHTTPClient()
	target := "https://example.com/article"
	text, err := shared.FetchTextContent(client, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/proxy" {
		t.Errorf("expected request path /proxy, got %q", gotPath)
	}
	if gotQuery != target {
		t.Errorf("expected url query %q, got %q", target, gotQuery)
	}
	if !strings.Contains(text, "hello world primary content here") {
		t.Errorf("expected extracted text to contain page content, got %q", text)
	}
}

func TestExtractPrimaryURL_UsesProxyEndpoint(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("url")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="https://github.com/foo/bar">primary</a></body></html>`))
	}))
	defer srv.Close()

	t.Setenv("BARE_WEB_PROXY_URL", srv.URL)

	client := shared.NewHTTPClient()
	target := "https://example.com/secondary"
	primary, err := shared.ExtractPrimaryURL(client, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/proxy" {
		t.Errorf("expected request path /proxy, got %q", gotPath)
	}
	if gotQuery != target {
		t.Errorf("expected url query %q, got %q", target, gotQuery)
	}
	if primary != "https://github.com/foo/bar" {
		t.Errorf("expected primary URL to be extracted, got %q", primary)
	}
}
