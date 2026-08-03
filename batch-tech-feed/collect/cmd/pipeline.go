package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

// RunPipeline は collect -> filter -> enrich -> compose を一括でシームレスに実行する。
func RunPipeline(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("pipeline", flag.ContinueOnError)
	sourcesPath := fs.String("sources", "/etc/tech-feed/sources.json", "path to sources.json")
	sinceDays := fs.Int("since-days", 2, "how far back to look")
	topN := fs.Int("top", 15, "number of candidates to filter")
	maxEnrich := fs.Int("max-enrich", 10, "max candidates for enrich")
	topics := fs.Int("topics", 3, "number of topics for script")
	digestDate := fs.String("digest-date", "", "YYYY-MM-DD (default: today)")
	outputPath := fs.String("output", "/tmp/script.json", "where to write final script.json")
	workDir := fs.String("work-dir", "/tmp", "working directory for intermediate json files")

	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	candFile := filepath.Join(*workDir, "candidates.json")
	filtFile := filepath.Join(*workDir, "filtered.json")
	enriFile := filepath.Join(*workDir, "enriched.json")

	log.Println("=== Step 1: Collecting candidates ===")
	collectArgs := []string{
		"-sources", *sourcesPath,
		"-since-days", fmt.Sprintf("%d", *sinceDays),
		"-output", candFile,
	}
	if err := RunCollect(ctx, collectArgs); err != nil {
		return fmt.Errorf("pipeline step 1 (collect) failed: %w", err)
	}

	log.Println("=== Step 2: Filtering trend candidates ===")
	filterArgs := []string{
		"-input", candFile,
		"-output", filtFile,
		"-top", fmt.Sprintf("%d", *topN),
	}
	if err := RunFilter(ctx, filterArgs); err != nil {
		return fmt.Errorf("pipeline step 2 (filter) failed: %w", err)
	}

	log.Println("=== Step 3: Enriching and verifying primary sources ===")
	enrichArgs := []string{
		"-input", filtFile,
		"-output", enriFile,
		"-max", fmt.Sprintf("%d", *maxEnrich),
	}
	if err := RunEnrich(ctx, enrichArgs); err != nil {
		return fmt.Errorf("pipeline step 3 (enrich) failed: %w", err)
	}

	log.Println("=== Step 4: Composing script ===")
	composeArgs := []string{
		"-input", enriFile,
		"-output", *outputPath,
		"-topics", fmt.Sprintf("%d", *topics),
	}
	if *digestDate != "" {
		composeArgs = append(composeArgs, "-digest-date", *digestDate)
	}
	if err := RunCompose(ctx, composeArgs); err != nil {
		return fmt.Errorf("pipeline step 4 (compose) failed: %w", err)
	}

	log.Printf("=== Pipeline completed successfully! Final script -> %s ===", *outputPath)
	return nil
}
