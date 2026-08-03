package shared

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var primaryDomainPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)github\.com/[^/]+/[^/]+`),
	regexp.MustCompile(`(?i)[a-z0-9-]+\.openai\.com`),
	regexp.MustCompile(`(?i)[a-z0-9-]+\.anthropic\.com`),
	regexp.MustCompile(`(?i)kubernetes\.io`),
	regexp.MustCompile(`(?i)cncf\.io`),
	regexp.MustCompile(`(?i)aws\.amazon\.com`),
	regexp.MustCompile(`(?i)blog\.cloudflare\.com`),
	regexp.MustCompile(`(?i)qwenlm\.github\.io`),
	regexp.MustCompile(`(?i)developer\.nvidia\.com`),
	regexp.MustCompile(`(?i)nvidianews\.nvidia\.com`),
	regexp.MustCompile(`(?i)ir\.amd\.com`),
	regexp.MustCompile(`(?i)newsroom\.intel\.com`),
}

// IsPrimaryDomain は与えられた URL が一次情報ドメインであるかを判定する。
func IsPrimaryDomain(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Host
	for _, pattern := range primaryDomainPatterns {
		if pattern.MatchString(host) || pattern.MatchString(rawURL) {
			return true
		}
	}
	return false
}

// ExtractPrimaryURL は二次情報のページ内から一次情報ソースと思われる URL を抽出する。
func ExtractPrimaryURL(client *http.Client, pageURL string) (string, error) {
	proxyBase := os.Getenv("BARE_WEB_PROXY_URL")
	fetchURL := pageURL
	if proxyBase != "" {
		fetchURL = fmt.Sprintf("%s/?url=%s", strings.TrimRight(proxyBase, "/"), url.QueryEscape(pageURL))
	}

	res, err := Get(client, fetchURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secondary page: %w", err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse secondary page HTML: %w", err)
	}

	var foundPrimary string
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if foundPrimary != "" {
			return
		}
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		// 絶対パスに変換
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		if !u.IsAbs() {
			base, err := url.Parse(pageURL)
			if err == nil {
				u = base.ResolveReference(u)
			}
		}

		fullURL := u.String()
		if IsPrimaryDomain(fullURL) {
			foundPrimary = fullURL
		}
	})

	if foundPrimary != "" {
		return foundPrimary, nil
	}

	return "", fmt.Errorf("no primary source URL found in %s", pageURL)
}

// FetchTextContent は URL からベタテキスト本文を取得する。
// bare-web-proxy が利用可能な場合はプロキシ経由で描画済みの主要テキストを取り出す。
func FetchTextContent(client *http.Client, targetURL string) (string, error) {
	proxyBase := os.Getenv("BARE_WEB_PROXY_URL")
	fetchURL := targetURL
	if proxyBase != "" {
		fetchURL = fmt.Sprintf("%s/?url=%s", strings.TrimRight(proxyBase, "/"), url.QueryEscape(targetURL))
	}

	res, err := Get(client, fetchURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		// HTMLでない場合は生のレスポンスを読む
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return Truncate(StripTags(string(bodyBytes)), 3000), nil
	}

	// 不要タグを削除
	doc.Find("script, style, nav, footer, header, iframe, noscript").Remove()

	// 主要テキストの抽出
	var textBlocks []string
	doc.Find("article, main, p, h1, h2, h3, li, blockquote").Each(func(_ int, sel *goquery.Selection) {
		t := strings.TrimSpace(sel.Text())
		if len(t) > 20 {
			textBlocks = append(textBlocks, t)
		}
	})

	if len(textBlocks) == 0 {
		// パースできない場合は全体のテキストを取得
		fullText := strings.TrimSpace(doc.Find("body").Text())
		return Truncate(StripTags(fullText), 3000), nil
	}

	joined := strings.Join(textBlocks, "\n")
	return Truncate(joined, 4000), nil
}
