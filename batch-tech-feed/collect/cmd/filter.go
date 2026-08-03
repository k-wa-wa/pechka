package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

// FilterResult は OpenAI が返してくる結果の型
type FilterResult struct {
	SelectedURLs []string `json:"selected_urls"`
}

// RunFilter は candidates.json を読み込み、話題性・関心度の高い上位 N 件を選別して filtered.json に書き出す。
func RunFilter(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("filter", flag.ContinueOnError)
	inputPath := fs.String("input", "/tmp/candidates.json", "path to candidates.json")
	outputPath := fs.String("output", "/tmp/filtered.json", "where to write filtered.json")
	topN := fs.Int("top", 15, "number of candidates to select")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		return fmt.Errorf("failed to read candidates: %w", err)
	}

	var candidates []Candidate
	if err := json.Unmarshal(raw, &candidates); err != nil {
		return fmt.Errorf("failed to parse candidates: %w", err)
	}

	log.Printf("filter: input candidates count = %d", len(candidates))
	if len(candidates) == 0 {
		return fmt.Errorf("input candidates list is empty")
	}

	selected := filterCandidates(ctx, candidates, *topN)
	log.Printf("filter: selected %d candidates -> %s", len(selected), *outputPath)

	body, err := json.MarshalIndent(selected, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode filtered candidates: %w", err)
	}

	return os.WriteFile(*outputPath, body, 0o644)
}

func filterCandidates(ctx context.Context, candidates []Candidate, topN int) []Candidate {
	aiClient := shared.NewOpenAIClient()
	if aiClient.IsConfigured() {
		log.Printf("filter: running LLM-based trend filter via OpenAI API...")
		filtered, err := filterWithLLM(ctx, aiClient, candidates, topN)
		if err == nil && len(filtered) > 0 {
			return filtered
		}
		log.Printf("filter: LLM filter failed or returned empty (%v); falling back to rule-based filter", err)
	} else {
		log.Printf("filter: OPENAI_API_KEY not set; using rule-based trend filter")
	}

	return filterRuleBased(candidates, topN)
}

func filterWithLLM(ctx context.Context, client *shared.OpenAIClient, candidates []Candidate, topN int) ([]Candidate, error) {
	// プロンプトが大きくなりすぎないよう、最大60件のリストを作成
	limit := 60
	if len(candidates) < limit {
		limit = len(candidates)
	}
	subList := candidates[:limit]

	var sb strings.Builder
	for i, c := range subList {
		scoreStr := ""
		if c.Score != nil {
			scoreStr = fmt.Sprintf(" [HN Score: %d]", *c.Score)
		}
		primaryStr := "[2次情報]"
		if c.IsPrimary {
			primaryStr = "[1次情報]"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s%s\n   URL: %s\n   From: %s\n   Summary: %s\n\n",
			i+1, primaryStr, c.Title, scoreStr, c.URL, c.Publisher, c.Summary))
	}

	prompt := fmt.Sprintf(`あなたはエンジニア向け技術ニュースの編集長です。
以下の候補記事リストから、日本のバックエンド・インフラ系エンジニアにとって「いま最も関心度・話題性が高く、技術的に面白くて実務に有益なトピック」を%d件選んでください。

# 選別方針:
- 2次情報（Hacker NewsやZennなど）で話題になっているトピックも、関心度が高ければ積極的に選んで構いません。
- 単なる宣伝や内容の薄い記事、似た話題の重複は避けてください。
- 一次情報の公式発表と、二次情報のトレンド解説のバランスを考慮してください。

# 出力フォーマット:
必ず以下の構造の JSON のみを出力してください。余計な解説文やコードブロックは付けないでください。
{
  "selected_urls": [
    "選んだ記事1のURL",
    "選んだ記事2のURL"
  ]
}

# 候補記事リスト:
%s`, topN, sb.String())

	respText, err := client.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// JSON抽出
	start := strings.Index(respText, "{")
	end := strings.LastIndex(respText, "}")
	if start < 0 || end < 0 || start >= end {
		return nil, fmt.Errorf("no valid JSON object found in response")
	}

	var res FilterResult
	if err := json.Unmarshal([]byte(respText[start:end+1]), &res); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w", err)
	}

	urlMap := make(map[string]Candidate)
	for _, c := range candidates {
		urlMap[c.URL] = c
	}

	var out []Candidate
	for _, urlStr := range res.SelectedURLs {
		if cand, ok := urlMap[urlStr]; ok {
			out = append(out, cand)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("none of the selected URLs matched original candidates")
	}

	return out, nil
}

func filterRuleBased(candidates []Candidate, topN int) []Candidate {
	// スコアが高いものを優先、また発信元ごとに偏りが出ないよう並べ替え
	cands := make([]Candidate, len(candidates))
	copy(cands, candidates)

	sort.SliceStable(cands, func(i, j int) bool {
		scoreI := 0
		scoreJ := 0
		if cands[i].Score != nil {
			scoreI = *cands[i].Score
		}
		if cands[j].Score != nil {
			scoreJ = *cands[j].Score
		}
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return cands[i].PublishedAt > cands[j].PublishedAt
	})

	if len(cands) > topN {
		return cands[:topN]
	}
	return cands
}
