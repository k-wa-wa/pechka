package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/k-wa-wa/pechka/batch-tech-feed/shared"
)

// RunEnrich は filtered.json を受け取り、一次情報の検証およびフル本文の取得を行って enriched.json に書き出す。
func RunEnrich(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("enrich", flag.ContinueOnError)
	inputPath := fs.String("input", "/tmp/filtered.json", "path to filtered.json")
	outputPath := fs.String("output", "/tmp/enriched.json", "where to write enriched.json")
	sourcesPath := fs.String("sources", "/etc/tech-feed/sources.json", "path to sources.json")
	maxOutput := fs.Int("max", 10, "maximum number of verified enriched candidates")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	if rawSources, err := os.ReadFile(*sourcesPath); err == nil {
		var sources Sources
		if err := json.Unmarshal(rawSources, &sources); err == nil {
			shared.SetPrimaryDomainPatterns(sources.PrimaryDomainPatterns)
		}
	}

	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var candidates []Candidate
	if err := json.Unmarshal(raw, &candidates); err != nil {
		return fmt.Errorf("failed to parse input candidates: %w", err)
	}

	log.Printf("enrich: processing %d candidates for primary source verification & content fetch...", len(candidates))
	httpClient := shared.NewHTTPClient()

	var enriched []Candidate
	for _, c := range candidates {
		if len(enriched) >= *maxOutput {
			break
		}

		if c.IsPrimary {
			log.Printf("  [Primary Source] Fetching content for %s (%s)...", c.Title, c.URL)
			content, err := shared.FetchTextContent(httpClient, c.URL)
			if err != nil {
				log.Printf("    WARN: failed to fetch content for %s: %v", c.URL, err)
				content = c.Summary
			}
			c.PrimaryURL = c.URL
			c.Content = content
			enriched = append(enriched, c)
		} else {
			log.Printf("  [Secondary Source] Tracing primary link for %s (%s)...", c.Title, c.URL)
			primaryURL, err := shared.ExtractPrimaryURL(httpClient, c.URL)
			if err != nil {
				log.Printf("    WARN: no valid primary source link found in secondary source (%s): %v", c.URL, err)
				// 誤情報防止のため、1次ソースが見つからない2次情報は落とす
				continue
			}

			log.Printf("    -> Found Primary URL: %s. Fetching content...", primaryURL)
			content, err := shared.FetchTextContent(httpClient, primaryURL)
			if err != nil {
				log.Printf("    WARN: failed to fetch content for primary URL (%s): %v", primaryURL, err)
				continue
			}

			c.PrimaryURL = primaryURL
			c.Content = content
			// 二次情報から見つかった一次情報として保持
			enriched = append(enriched, c)
		}
	}

	log.Printf("enrich: successfully verified & enriched %d candidates -> %s", len(enriched), *outputPath)
	if len(enriched) == 0 {
		return fmt.Errorf("no verified candidates remained after enrichment")
	}

	body, err := json.MarshalIndent(enriched, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode enriched candidates: %w", err)
	}

	return os.WriteFile(*outputPath, body, 0o644)
}
