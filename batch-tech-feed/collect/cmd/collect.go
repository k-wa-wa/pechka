package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

const (
	hnAPI     = "https://hacker-news.firebaseio.com/v0"
	githubAPI = "https://api.github.com"
	// フィードが配る要約だけを持つ。記事本文は取りに行かない(docs/407 §2.1)。
	summaryLimit = 400
)

// Candidate は Python 側(techfeed/candidates.py)および後続フェーズと対になる。
type Candidate struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Publisher   string `json:"publisher"`
	PublishedAt string `json:"published_at"`
	Summary     string `json:"summary"`
	Score       *int   `json:"score,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
	PrimaryURL  string `json:"primary_url,omitempty"`
	Content     string `json:"content,omitempty"`
}

// HTMLScrapeSource は HTML スクレイピング対象の設定。
type HTMLScrapeSource struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	IsDynamic         bool   `json:"is_dynamic"`
	IsPrimary         bool   `json:"is_primary"`
	ContainerSelector string `json:"container_selector"`
	TitleSelector     string `json:"title_selector"`
	URLAttribute      string `json:"url_attribute"`
	BaseURL           string `json:"base_url"`
}

// GitHubReleaseSource は GitHub Releases のソース設定。
type GitHubReleaseSource struct {
	Repo      string `json:"repo"`
	IsPrimary bool   `json:"is_primary"`
}

// Sources は sources.json の形。
type Sources struct {
	PrimaryDomainPatterns []string `json:"primary_domain_patterns"`
	RSS                   []struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		IsPrimary bool   `json:"is_primary"`
	} `json:"rss"`
	HackerNews struct {
		Enabled   bool `json:"enabled"`
		TopN      int  `json:"top_n"`
		MinScore  int  `json:"min_score"`
		IsPrimary bool `json:"is_primary"`
	} `json:"hacker_news"`
	GitHubReleases []GitHubReleaseSource `json:"github_releases"`
	HTMLScrape     []HTMLScrapeSource    `json:"html_scrape"`
}

type collector struct {
	client *http.Client
	// この時刻より新しい記事だけを拾う。
	since time.Time
}

func (c *collector) within(published *time.Time) bool {
	return published == nil || published.After(c.since)
}

func iso(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (c *collector) fromRSS(name, feedURL string, isPrimary bool) ([]Candidate, error) {
	res, err := shared.Get(c.client, feedURL)
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
			IsPrimary:   isPrimary,
		})
	}
	return out, nil
}

func (c *collector) fromHackerNews(topN, minScore int, isPrimary bool) ([]Candidate, error) {
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
		if err := shared.GetJSON(c.client, fmt.Sprintf("%s/item/%d.json", hnAPI, id), &item); err != nil {
			continue
		}
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
			IsPrimary:   isPrimary,
		})
	}
	return out, nil
}

func (c *collector) fromGitHubReleases(repo string, isPrimary bool) ([]Candidate, error) {
	var releases []struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		PublishedAt string `json:"published_at"`
	}
	reqURL := fmt.Sprintf("%s/repos/%s/releases?per_page=5", githubAPI, repo)
	if err := shared.GetJSON(c.client, reqURL, &releases); err != nil {
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
			IsPrimary:   isPrimary,
		})
	}
	return out, nil
}

func (c *collector) fromHTMLScrape(s HTMLScrapeSource) ([]Candidate, error) {
	targetURL := s.URL
	proxyBase := os.Getenv("BARE_WEB_PROXY_URL")
	if s.IsDynamic && proxyBase != "" {
		targetURL = fmt.Sprintf("%s/proxy?url=%s", strings.TrimRight(proxyBase, "/"), url.QueryEscape(s.URL))
	}

	res, err := shared.Get(c.client, targetURL)
	if err != nil {
		return nil, fmt.Errorf("fetch html failed for %s (url: %s): %w", s.Name, targetURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch html HTTP %d for %s (url: %s)", res.StatusCode, s.Name, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html failed for %s: %w", s.Name, err)
	}

	var out []Candidate
	seen := make(map[string]bool)

	matched := doc.Find(s.ContainerSelector)
	if matched.Length() == 0 {
		log.Printf("  html %-28s WARN: selector %q matched 0 elements (target: %s)", s.Name, s.ContainerSelector, targetURL)
	}

	matched.Each(func(_ int, sel *goquery.Selection) {
		link, exists := sel.Attr(s.URLAttribute)
		if !exists || link == "" {
			return
		}
		if s.BaseURL != "" && !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			link = strings.TrimRight(s.BaseURL, "/") + "/" + strings.TrimLeft(link, "/")
		}

		if strings.Contains(link, "/proxy?url=") || strings.Contains(link, "/proxy?q=") {
			if u, err := url.Parse(link); err == nil {
				if q := u.Query().Get("url"); q != "" {
					link = q
				} else if q := u.Query().Get("q"); q != "" {
					link = q
				}
			}
		}

		if seen[link] {
			return
		}
		seen[link] = true

		title := ""
		if s.TitleSelector != "" {
			title = strings.TrimSpace(sel.Find(s.TitleSelector).First().Text())
		}
		if title == "" {
			title = strings.TrimSpace(sel.Text())
		}
		if title == "" || len(title) < 5 {
			return
		}

		now := time.Now().UTC()
		out = append(out, Candidate{
			Title:       title,
			URL:         link,
			Publisher:   s.Name,
			PublishedAt: iso(&now),
			Summary:     shared.Truncate(title, summaryLimit),
			IsPrimary:   s.IsPrimary,
		})
	})
	return out, nil
}

func (c *collector) collectAll(sources Sources) []Candidate {
	var found []Candidate

	for _, feed := range sources.RSS {
		got, err := c.fromRSS(feed.Name, feed.URL, feed.IsPrimary)
		if err != nil {
			log.Printf("  rss  %-28s SKIP (%v)", feed.Name, err)
			continue
		}
		log.Printf("  rss  %-28s %3d", feed.Name, len(got))
		found = append(found, got...)
	}

	if sources.HackerNews.Enabled {
		got, err := c.fromHackerNews(sources.HackerNews.TopN, sources.HackerNews.MinScore, sources.HackerNews.IsPrimary)
		if err != nil {
			log.Printf("  hn   %-28s SKIP (%v)", "Hacker News", err)
		} else {
			log.Printf("  hn   %-28s %3d", "Hacker News", len(got))
			found = append(found, got...)
		}
	}

	for _, gh := range sources.GitHubReleases {
		got, err := c.fromGitHubReleases(gh.Repo, gh.IsPrimary)
		if err != nil {
			log.Printf("  gh   %-28s SKIP (%v)", gh.Repo, err)
			continue
		}
		log.Printf("  gh   %-28s %3d", gh.Repo, len(got))
		found = append(found, got...)
	}

	for _, scrape := range sources.HTMLScrape {
		got, err := c.fromHTMLScrape(scrape)
		if err != nil {
			log.Printf("  html %-28s SKIP (%v)", scrape.Name, err)
			continue
		}
		log.Printf("  html %-28s %3d", scrape.Name, len(got))
		found = append(found, got...)
	}

	return found
}

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
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PublishedAt > out[j].PublishedAt
	})
	return out
}

// RunCollect は情報源から候補を集め、candidates.json として書き出す。
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
	shared.SetPrimaryDomainPatterns(sources.PrimaryDomainPatterns)

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
		return fmt.Errorf("no candidates collected; check %s and the network", *sourcesPath)
	}
	return nil
}
