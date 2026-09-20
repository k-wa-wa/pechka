package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// longArticleHTML は妥当性チェックを通過するのに十分な長さの本文を持つページ。
func longArticleHTML(marker string) string {
	paragraph := strings.Repeat(marker+"についての詳細な技術的な説明が続きます。", 5)
	return fmt.Sprintf(`<html><body><article><h1>%s</h1><p>%s</p></article></body></html>`, marker, paragraph)
}

// tooShortHTML は本文が極端に短く、妥当性チェックで弾かれるべきページ。
func tooShortHTML() string {
	return `<html><body><p>短い</p></body></html>`
}

// landingPageHTML は bare-web-proxy のボット判定/エラーページを模したもの。
func landingPageHTML() string {
	filler := strings.Repeat("checking your browser before accessing this site. ", 5)
	return fmt.Sprintf(`<html><body><main><p>%s</p></main></body></html>`, filler)
}

// secondaryPageHTML は一次情報へのリンクを含む二次情報ページ。
func secondaryPageHTML(primaryURL string) string {
	return fmt.Sprintf(`<html><body><article><p>まとめ記事の本文です。</p><a href="%s">元記事はこちら</a></article></body></html>`, primaryURL)
}

func TestRunEnrich_ValidatesFetchedContent(t *testing.T) {
	const (
		primaryValidURL     = "https://github.com/example/valid-repo"
		primaryTooShortURL  = "https://github.com/example/too-short-repo"
		secondaryURL        = "https://example.com/secondary-summary"
		secondaryPrimaryURL = "https://github.com/example/landing-repo"
	)

	pages := map[string]string{
		primaryValidURL:     longArticleHTML("valid-content"),
		primaryTooShortURL:  tooShortHTML(),
		secondaryURL:        secondaryPageHTML(secondaryPrimaryURL),
		secondaryPrimaryURL: landingPageHTML(),
	}

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("url")
		body, ok := pages[target]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	defer proxy.Close()

	t.Setenv("BARE_WEB_PROXY_URL", proxy.URL)

	candidates := []Candidate{
		{
			Title:     "Valid primary source",
			URL:       primaryValidURL,
			Publisher: "Test",
			Summary:   "valid summary",
			IsPrimary: true,
		},
		{
			Title:     "Too-short primary source",
			URL:       primaryTooShortURL,
			Publisher: "Test",
			Summary:   "fallback summary for too-short content",
			IsPrimary: true,
		},
		{
			Title:     "Secondary source with landing-page primary",
			URL:       secondaryURL,
			Publisher: "Test",
			Summary:   "secondary summary",
			IsPrimary: false,
		},
	}

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "filtered.json")
	outputPath := filepath.Join(dir, "enriched.json")
	sourcesPath := filepath.Join(dir, "does-not-exist-sources.json")

	raw, err := json.Marshal(candidates)
	if err != nil {
		t.Fatalf("failed to marshal candidates: %v", err)
	}
	if err := os.WriteFile(inputPath, raw, 0o644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	args := []string{
		"-input", inputPath,
		"-output", outputPath,
		"-sources", sourcesPath,
	}
	if err := RunEnrich(context.Background(), args); err != nil {
		t.Fatalf("RunEnrich returned error: %v", err)
	}

	outRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	var out []Candidate
	if err := json.Unmarshal(outRaw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	byURL := make(map[string]Candidate)
	for _, c := range out {
		byURL[c.URL] = c
	}

	// 妥当なコンテンツはそのまま使われる。
	valid, ok := byURL[primaryValidURL]
	if !ok {
		t.Fatalf("expected valid primary candidate to remain, got: %+v", out)
	}
	if !strings.Contains(valid.Content, "valid-content") {
		t.Errorf("expected fetched content to be used for valid primary source, got: %q", valid.Content)
	}

	// 極端に短いコンテンツは無言で通さず、要約にフォールバックする。
	tooShort, ok := byURL[primaryTooShortURL]
	if !ok {
		t.Fatalf("expected too-short primary candidate to remain with fallback, got: %+v", out)
	}
	if tooShort.Content != tooShort.Summary {
		t.Errorf("expected too-short content to fall back to summary, got content: %q", tooShort.Content)
	}

	// 二次情報から辿った一次情報がランディングページ特徴を含む場合は候補ごと除外する。
	if _, ok := byURL[secondaryURL]; ok {
		t.Errorf("expected secondary candidate with invalid primary content to be excluded from output")
	}
}
