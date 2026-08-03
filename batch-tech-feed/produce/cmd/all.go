package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

// RunAll は synthesize -> render -> publish を一括でシームレスに実行する。
func RunAll(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	scriptPath := fs.String("script", "", "path to script.json")
	outDir := fs.String("out", "", "working directory for intermediate artifacts")
	engine := fs.String("engine", "voicevox", "engine type: voicevox or mock")
	speaker := fs.Int("speaker", shared.DefaultSpeakerID, "speaker ID")
	sourceKey := fs.String("source-key", "", "idempotency key for re-runs")
	quiet := fs.Bool("quiet", false, "suppress progress output")

	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	if *scriptPath == "" || *outDir == "" {
		return fmt.Errorf("--script and --out parameters are required")
	}

	// 1. Synthesize
	synthArgs := []string{
		"-script", *scriptPath,
		"-out", *outDir,
		"-engine", *engine,
		"-speaker", fmt.Sprintf("%d", *speaker),
	}
	if _, err := RunSynthesize(ctx, synthArgs); err != nil {
		return fmt.Errorf("all: synthesize step failed: %w", err)
	}

	// 2. Render
	renderArgs := []string{
		"-out", *outDir,
	}
	if *quiet {
		renderArgs = append(renderArgs, "-quiet")
	}
	if err := RunRender(ctx, renderArgs); err != nil {
		return fmt.Errorf("all: render step failed: %w", err)
	}

	// 3. Publish
	publishArgs := []string{
		"-out", *outDir,
	}
	if *sourceKey != "" {
		publishArgs = append(publishArgs, "-source-key", *sourceKey)
	}
	if err := RunPublish(ctx, publishArgs); err != nil {
		return fmt.Errorf("all: publish step failed: %w", err)
	}

	return nil
}
