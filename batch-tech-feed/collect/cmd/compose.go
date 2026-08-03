package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

const maxAttempts = 3

// RunCompose は enriched.json を読み込み、LLM により検証済み台本 script.json を生成して書き出す。
func RunCompose(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("compose", flag.ContinueOnError)
	inputPath := fs.String("input", "/tmp/enriched.json", "path to enriched.json")
	outputPath := fs.String("output", "/tmp/script.json", "where to write script.json")
	topics := fs.Int("topics", 3, "how many topics to cover in the script")
	digestDate := fs.String("digest-date", "", "YYYY-MM-DD (default: today)")
	promptPath := fs.String("prompt", "/etc/tech-feed/prompt.txt", "optional path to custom prompt text")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	var customPrompt string
	if *promptPath != "" {
		if b, err := os.ReadFile(*promptPath); err == nil {
			customPrompt = strings.TrimSpace(string(b))
		}
	}

	if *digestDate == "" {
		*digestDate = time.Now().Format("2006-01-02")
	}

	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var candidates []Candidate
	if err := json.Unmarshal(raw, &candidates); err != nil {
		return fmt.Errorf("failed to parse enriched candidates: %w", err)
	}

	if len(candidates) == 0 {
		return fmt.Errorf("input enriched candidates list is empty")
	}

	log.Printf("compose: composing script from %d enriched candidate(s)...", len(candidates))

	allowedURLs := make(map[string]bool)
	for _, c := range candidates {
		targetURL := c.PrimaryURL
		if targetURL == "" {
			targetURL = c.URL
		}
		allowedURLs[targetURL] = true
	}

	client := shared.NewOpenAIClient()
	if !client.IsConfigured() {
		return fmt.Errorf("OPENAI_API_KEY must be configured for compose command")
	}

	basePrompt := buildComposePrompt(candidates, *digestDate, *topics, customPrompt)

	var lastError string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ask := basePrompt
		if attempt > 1 {
			ask = fmt.Sprintf("%s\n\n# 直前の出力は次の理由で不正だった。修正して JSON を出し直せ\n%s", basePrompt, lastError)
		}

		log.Printf("  attempt %d/%d...", attempt, maxAttempts)
		rawResp, err := client.Complete(ctx, ask)
		if err != nil {
			lastError = fmt.Sprintf("LLM completion error: %v", err)
			log.Printf("    rejected: %s", lastError)
			continue
		}

		script, err := parseAndValidateScript(rawResp, allowedURLs)
		if err != nil {
			lastError = fmt.Sprintf("Validation error: %v", err)
			log.Printf("    rejected: %s", lastError)
			continue
		}

		// 成功
		body, err := json.MarshalIndent(script, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal script JSON: %w", err)
		}

		if err := os.WriteFile(*outputPath, body, 0o644); err != nil {
			return fmt.Errorf("failed to write script to %s: %w", *outputPath, err)
		}

		log.Printf("compose: successfully generated script (%d sections) -> %s", len(script.Sections), *outputPath)
		return nil
	}

	return fmt.Errorf("compose failed after %d attempts. last error: %s", maxAttempts, lastError)
}

func buildComposePrompt(candidates []Candidate, digestDate string, topics int, customPrompt string) string {
	var sb strings.Builder
	for i, c := range candidates {
		primaryURL := c.PrimaryURL
		if primaryURL == "" {
			primaryURL = c.URL
		}
		content := c.Content
		if content == "" {
			content = c.Summary
		}
		// 1件あたりのコンテンツ長を制限してプロンプトサイズを適正に保つ
		content = shared.Truncate(content, 1200)

		sb.WriteString(fmt.Sprintf("%d. %s\n   url: %s\n   from: %s\n   本文:\n   %s\n\n",
			i+1, c.Title, primaryURL, c.Publisher, content))
	}

	rolePrompt := "あなたは技術ダイジェスト動画の構成作家である。"
	if customPrompt != "" {
		rolePrompt = customPrompt
	}

	return fmt.Sprintf(`%s 以下の検証済み一次情報候補から今日取り上げる%d件を選び、解説動画の台本を JSON で書け。

# 選別の基準
- 実務に影響するもの、一次情報の確証があるものを優先する
- 単なる宣伝、内容の薄いもの、同じ話題の重複は落とす
- 読者はインフラ・バックエンドを触る個人開発者である

# 出力する JSON の形式

{
  "digest_date": "%s",
  "title": "今日の技術トピック %d選",
  "sections": [ ...セクション... ]
}

セクションは次の形。1つ目は必ず layout="title" の表紙にし、以降が各トピック。

{
  "seq": 2,
  "slide": {
    "layout": "bullets",          // title | bullets | code | diagram のいずれか
    "title": "見出し（20字程度）",
    "subtitle": "補足（30字程度、任意）",
    "items": ["要点1", "要点2", "要点3"]   // bullets のとき必須。3件前後
    // code のとき: "code": "...", "language": "yaml"
    // diagram のとき: "diagram": "flowchart LR\n  A[\"箱\"] --> B[\"箱\"]"
  },
  "narration": [
    { "text": "読み上げる1文。", "focus": null },
    { "text": "この文は項目1の話。", "focus": 0 }
  ],
  "sources": [
    { "title": "記事タイトル", "url": "https://...", "publisher": "掲載元" }
  ]
}

# 厳守すること
- **narration は1文ずつに分ける。** 音声合成・字幕・スライド遷移の単位がこれである
- focus は bullets のときだけ使う。items の index（0始まり）を指し、null は「スライドを進めず前の状態のまま」を意味する。1つの項目を2文で語るなら同じ focus を2回書く
- focus に items の範囲外を書かない
- **sources には候補記事の url をそのまま入れる。URL を創作しない**
- 記事本文を丸ごと引き写さない。見出しと自分の言葉による要約に留める
- 図が説明に効く場合は diagram（Mermaid の flowchart）を使う。ノードのラベルは必ず二重引用符で囲む
- title レイアウトの表紙には narration を2文程度、sources は不要
- 全体で90秒〜3分程度の分量にする
- **JSON のみを出力する。前置き・説明・コードフェンスを付けない**

# 候補記事
%s`, rolePrompt, topics, digestDate, topics, sb.String())
}

func parseAndValidateScript(raw string, allowedURLs map[string]bool) (*shared.Script, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < 0 || start >= end {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	jsonStr := raw[start : end+1]
	var script shared.Script
	if err := json.Unmarshal([]byte(jsonStr), &script); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	if err := shared.ValidateScript(&script, allowedURLs); err != nil {
		return nil, err
	}

	return &script, nil
}
