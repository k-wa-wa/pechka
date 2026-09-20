package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// MinContentLength は本文として有効とみなす最小文字数（rune 数）。
// これを下回る場合、プロキシのエラー応答や空応答である可能性が高いとみなす。
const MinContentLength = 100

// suspiciousContentPatterns は、bare-web-proxy 自身のツールバー/エラー表示や、
// 対象サイトのボット判定・アクセス制限ページなど、本文として扱うべきでない
// 既知のランディングページの特徴を検出するためのパターン。
var suspiciousContentPatterns = []*regexp.Regexp{
	// bare-web-proxy 自身が返しうるエラー文言・UI要素
	regexp.MustCompile(`(?i)bare-web-proxy`),
	regexp.MustCompile(`(?i)failed to fetch`),
	regexp.MustCompile(`(?i)proxy error`),
	// Cloudflare 等のボット判定・チャレンジページ
	regexp.MustCompile(`(?i)checking your browser before accessing`),
	regexp.MustCompile(`(?i)just a moment\.\.\.`),
	regexp.MustCompile(`(?i)enable javascript and cookies to continue`),
	regexp.MustCompile(`(?i)verify (that )?you are a human`),
	regexp.MustCompile(`(?i)attention required.{0,20}cloudflare`),
	// 一般的なアクセス拒否・404ページ
	regexp.MustCompile(`(?i)access denied`),
	regexp.MustCompile(`(?i)403 forbidden`),
	regexp.MustCompile(`(?i)404 not found`),
	regexp.MustCompile(`(?i)page not found`),
}

// ValidateContent は FetchTextContent 等で取得した本文が、後段の要約・台本
// 生成に使うに足る妥当なものかを検証する。空・極端に短いコンテンツや、
// プロキシ/対象サイトのエラーページ・ボット判定ページと思われるコンテンツを
// 検出した場合はエラーを返す。呼び出し元はこれを候補除外の判断に使う。
func ValidateContent(content string) error {
	trimmed := strings.TrimSpace(content)

	if length := len([]rune(trimmed)); length < MinContentLength {
		return fmt.Errorf("content too short (%d runes, minimum %d)", length, MinContentLength)
	}

	for _, pattern := range suspiciousContentPatterns {
		if pattern.MatchString(trimmed) {
			return fmt.Errorf("content matched suspicious pattern %q", pattern.String())
		}
	}

	return nil
}
