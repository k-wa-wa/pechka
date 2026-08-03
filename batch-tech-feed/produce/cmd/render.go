package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

func RunRender(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	outDir := fs.String("out", "", "working directory")
	output := fs.String("output", "", "output mp4 path (default: <out>/digest.mp4)")
	crf := fs.Int("crf", 20, "CRF quality")
	concurrency := fs.Int("concurrency", 0, "rendering concurrency")
	quiet := fs.Bool("quiet", false, "suppress remotion progress output")

	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	if *outDir == "" {
		return fmt.Errorf("--out parameter is required")
	}

	manifestPath := filepath.Join(*outDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("%s not found; run the 'synthesize' step first", manifestPath)
	}

	var manifest shared.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	dst := *output
	if dst == "" {
		dst = filepath.Join(*outDir, "digest.mp4")
	}

	fmt.Printf("rendering with remotion (%d line(s))...\n", len(manifest.Entries))
	renderedDst, err := shared.RunRender(&manifest, *outDir, dst, *concurrency, *crf, *quiet)
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	fmt.Printf("done: %s\n", renderedDst)
	return nil
}
