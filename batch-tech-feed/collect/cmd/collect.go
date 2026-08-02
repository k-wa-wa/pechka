package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

const (
	hnAPI     = "https://hacker-news.firebaseio.com/v0"
	githubAPI = "https://api.github.com"
	// フィードが配る要約だけを持つ。記事本文は取りに行かない(docs/407 §2.1)。
	summaryLimit = 400
)

// Candidate は Python 側(techfeed/candidates.py)の dataclass と対になる。
// フィールド名を変えるときは両方を直すこと。
type Candidate struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Publisher   string `json:"publisher"`
	PublishedAt string `json:"published_at"`
	Summary     string `json:"summary"`
	Score       *int   `json:"score"`
}

// Sources は sources.json の形。
type Sources struct {
	RSS []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"rss"`
	HackerNews struct {
		Enabled  bool `json:"enabled"`
		TopN     int  `json:"top_n"`
		MinScore int  `json:"min_score"`
	} `json:"hacker_news"`
	// 未認証でも叩けるが、GitHub API はレート制限が 60 req/hour と低い。
	// 対象を増やすなら認証を足すこと。
	GitHubReleases []string `json:"github_releases"`
}

type collector struct {
	client *http.Client
	// この時刻より新しい記事だけを拾う。
	since time.Time
}

// within は日付が読めないフィードを弾かずに通す。落とすより拾いすぎる方が害が小さい。
func (c *collector) within(published *time.Time) bool {
	return published == nil || published.After(c.since)
}

func iso(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (c *collector) fromRSS(name, url string) ([]Candidate, error) {
	res, err := shared.Get(c.client, url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	feed, err := gofeed.NewParser().Parse(res.Body)
	if err != nil {
		return nil, err
	}

	var out []Candidate
	for _, item := range feed.Items {
		published := item.PublishedParsed
		if published == nil {
			published = item.UpdatedParsed
		}
		if !c.within(published) || item.Title == "" || item.Link == "" {
			continue
		}
		out = append(out, Candidate{
			Title:       strings.TrimSpace(item.Title),
			URL:         item.Link,
			Publisher:   name,
			PublishedAt: iso(published),
			Summary:     shared.Truncate(shared.StripTags(item.Description), summaryLimit),
		})
	}
	return out, nil
}

func (c *collector) fromHackerNews(topN, minScore int) ([]Candidate, error) {
	var ids []int
	if err := shared.GetJSON(c.client, hnAPI+"/topstories.json", &ids); err != nil {
		return nil, err
	}
	if len(ids) > topN {
		ids = ids[:topN]
	}

	var out []Candidate
	for _, id := range ids {
		var item struct {
			Type  string `json:"type"`
			Title string `json:"title"`
			URL   string `json:"url"`
			Score int    `json:"score"`
			Time  int64  `json:"time"`
		}
		// 1件取れなくても残りで続ける。1件の失敗で収集全体を落とす価値はない。
		if err := shared.GetJSON(c.client, fmt.Sprintf("%s/item/%d.json", hnAPI, id), &item); err != nil {
			continue
		}
		// Ask HN 等は URL を持たない。動画で紹介できないので落とす。
		if item.Type != "story" || item.URL == "" || item.Score < minScore {
			continue
		}
		published := time.Unix(item.Time, 0).UTC()
		if !c.within(&published) {
			continue
		}
		score := item.Score
		out = append(out, Candidate{
			Title:       strings.TrimSpace(item.Title),
			URL:         item.URL,
			Publisher:   "Hacker News",
			PublishedAt: iso(&published),
			Score:       &score,
		})
	}
	return out, nil
}

func (c *collector) fromGitHubReleases(repo string) ([]Candidate, error) {
	var releases []struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		PublishedAt string `json:"published_at"`
	}
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=5", githubAPI, repo)
	if err := shared.GetJSON(c.client, url, &releases); err != nil {
		return nil, err
	}

	var out []Candidate
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		published, err := time.Parse(time.RFC3339, rel.PublishedAt)
		if err != nil || !c.within(&published) {
			continue
		}
		out = append(out, Candidate{
			Title:       strings.TrimSpace(repo + " " + rel.TagName),
			URL:         rel.HTMLURL,
			Publisher:   "GitHub / " + repo,
			PublishedAt: iso(&published),
			Summary:     shared.Truncate(rel.Body, summaryLimit),
		})
	}
	return out, nil
}

// collectAll は情報源を順に巡り、集まった候補を返す。
// 1つの情報源が落ちていても残りで続行する(docs/102 US-6.2)。
func (c *collector) collectAll(sources Sources) []Candidate {
	var found []Candidate

	for _, feed := range sources.RSS {
		got, err := c.fromRSS(feed.Name, feed.URL)
		if err != nil {
			log.Printf("  rss  %-28s SKIP (%v)", feed.Name, err)
			continue
		}
		log.Printf("  rss  %-28s %3d", feed.Name, len(got))
		found = append(found, got...)
	}

	if sources.HackerNews.Enabled {
		got, err := c.fromHackerNews(sources.HackerNews.TopN, sources.HackerNews.MinScore)
		if err != nil {
			log.Printf("  hn   %-28s SKIP (%v)", "Hacker News", err)
		} else {
			log.Printf("  hn   %-28s %3d", "Hacker News", len(got))
			found = append(found, got...)
		}
	}

	for _, repo := range sources.GitHubReleases {
		got, err := c.fromGitHubReleases(repo)
		if err != nil {
			log.Printf("  gh   %-28s SKIP (%v)", repo, err)
			continue
		}
		log.Printf("  gh   %-28s %3d", repo, len(got))
		found = append(found, got...)
	}
	return found
}

// dedupe は同じ URL を1件に畳み、新しい順に並べる。
// 複数のフィードに同じ記事が載ることがあるため。
func dedupe(found []Candidate) []Candidate {
	seen := map[string]bool{}
	out := make([]Candidate, 0, len(found))
	for _, cand := range found {
		if cand.URL == "" || cand.Title == "" || seen[cand.URL] {
			continue
		}
		seen[cand.URL] = true
		out = append(out, cand)
	}
	// プロンプトの先頭ほど目に入りやすいので、新しいものを前に置く。
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PublishedAt > out[j].PublishedAt
	})
	return out
}

// RunCollect は情報源から候補を集め、candidates.json として書き出す。
//
// 出力先を /tmp 配下のファイルにしているのは、Argo の outputs.parameters で
// 次のステップへ渡すためである(Bluray 取り込みが mkv-files.json / short-id を
// 受け渡しているのと同じ形)。この工程は外部ストレージも DB も触らないので、
// 認証情報を一切必要としない。
func RunCollect(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	sourcesPath := fs.String("sources", "/etc/tech-feed/sources.json", "path to sources.json")
	sinceDays := fs.Int("since-days", 2, "how far back to look")
	output := fs.String("output", "/tmp/candidates.json", "where to write candidates.json")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	raw, err := os.ReadFile(*sourcesPath)
	if err != nil {
		return fmt.Errorf("failed to read sources: %w", err)
	}
	var sources Sources
	if err := json.Unmarshal(raw, &sources); err != nil {
		return fmt.Errorf("failed to parse sources: %w", err)
	}

	c := &collector{
		client: shared.NewHTTPClient(),
		since:  time.Now().UTC().AddDate(0, 0, -*sinceDays),
	}
	candidates := dedupe(c.collectAll(sources))

	body, err := json.MarshalIndent(candidates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode candidates: %w", err)
	}
	if err := os.WriteFile(*output, body, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", *output, err)
	}

	log.Printf("candidates: %d -> %s", len(candidates), *output)
	if len(candidates) == 0 {
		// 全滅は情報源の設定ミスか、全サイトが落ちているかのどちらか。
		// 後段の台本生成が確実に失敗するので、ここで気づけるようにする。
		return fmt.Errorf("no candidates collected; check %s and the network", *sourcesPath)
	}
	return nil
}
