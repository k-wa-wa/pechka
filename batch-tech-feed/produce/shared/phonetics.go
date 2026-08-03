package shared

import (
	"regexp"
)

type termReplacement struct {
	re          *regexp.Regexp
	replacement string
}

var techTermReplacements []termReplacement
var versionPattern = regexp.MustCompile(`(?i)\bv(\d+(\.\d+)*)\b`)

func init() {
	rawTerms := []struct {
		pattern     string
		replacement string
	}{
		{`(?i)\bKubernetes\b`, "クーバネティス"},
		{`(?i)\bk8s\b`, "クーバネティス"},
		{`(?i)\bPostgreSQL\b`, "ポスグレエスケイエル"},
		{`(?i)\bPostgres\b`, "ポスグレ"},
		{`(?i)\bTypeScript\b`, "タイプスクリプト"},
		{`(?i)\bJavaScript\b`, "ジャバスクリプト"},
		{`(?i)\bNext\.js\b`, "ネクストジェイエス"},
		{`(?i)\bNode\.js\b`, "ノードジェイエス"},
		{`(?i)\bReact\b`, "リアクト"},
		{`(?i)\bPython\b`, "パイソン"},
		{`(?i)\bGolang\b`, "ゴーラング"},
		{`(?i)\bDocker\b`, "ドッカー"},
		{`(?i)\bRust\b`, "ラスト"},
		{`(?i)\bGitHub\b`, "ギットハブ"},
		{`(?i)\bGitLab\b`, "ギットラボ"},
		{`(?i)\bGit\b`, "ギット"},
		{`(?i)\bAWS\b`, "エーダブリューエス"},
		{`(?i)\bGCP\b`, "ジーシーピー"},
		{`(?i)\bAzure\b`, "アジュール"},
		{`(?i)\bCI/CD\b`, "シーアイシーディー"},
		{`(?i)\bAPI\b`, "エイピーアイ"},
		{`(?i)\bAPIs\b`, "エイピーアイズ"},
		{`(?i)\bSDK\b`, "エスディーケー"},
		{`(?i)\bSDKs\b`, "エスディーケーズ"},
		{`(?i)\bRSS\b`, "アールエスエス"},
		{`(?i)\bPR\b`, "ピーアール"},
		{`(?i)\bPRs\b`, "ピーアールズ"},
		{`(?i)\bIssue\b`, "イシュー"},
		{`(?i)\bIssues\b`, "イシューズ"},
		{`(?i)\bGA\b`, "ジーエー"},
		{`(?i)\bUI\b`, "ユーアイ"},
		{`(?i)\bUX\b`, "ユーエックス"},
		{`(?i)\bURL\b`, "ユーアールエル"},
		{`(?i)\bURLs\b`, "ユーアールエルズ"},
		{`(?i)\bHTTP\b`, "エイチティーティーピー"},
		{`(?i)\bHTTPS\b`, "エイチティーティーピーエス"},
		{`(?i)\bLLM\b`, "エルエルエム"},
		{`(?i)\bLLMs\b`, "エルエルエムズ"},
		{`(?i)\bTTS\b`, "ティーティーエス"},
		{`(?i)\bCPU\b`, "シーピーユー"},
		{`(?i)\bGPU\b`, "ジーピーユー"},
		{`(?i)\bAI\b`, "エーアイ"},
		{`(?i)\bOSS\b`, "オーエスエス"},
		{`(?i)\bREADME\b`, "リードミー"},
		{`(?i)\bDB\b`, "ディービー"},
		{`(?i)\bJSON\b`, "ジェイソン"},
		{`(?i)\bYAML\b`, "ヤムル"},
		{`(?i)\bHTML\b`, "エイチティーエムエル"},
		{`(?i)\bCSS\b`, "シーエスエス"},
		{`(?i)\bSQL\b`, "エスケイエル"},
	}

	for _, item := range rawTerms {
		techTermReplacements = append(techTermReplacements, termReplacement{
			re:          regexp.MustCompile(item.pattern),
			replacement: item.replacement,
		})
	}
}

// NormalizeForTTS はテキストをVOICEVOX読み上げ用に正規化する。
func NormalizeForTTS(text string) string {
	if text == "" {
		return text
	}

	result := versionPattern.ReplaceAllString(text, "バージョン ${1}")

	for _, tr := range techTermReplacements {
		result = tr.re.ReplaceAllString(result, tr.replacement)
	}

	return result
}
