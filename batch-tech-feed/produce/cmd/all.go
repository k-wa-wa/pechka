package cmd

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

// RunAll は Python 版 main.py の cmd_all と完全互換のフラグを受け取り、
// synthesize -> render -> publish を一括で順次実行する。
func RunAll(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)

	// Common / TTS / Render / Publish の全パラメータ
	scriptPath := fs.String("script", "", "path to script.json")
	outDir := fs.String("out", "", "working directory for intermediate artifacts")
	fps := fs.Int("fps", 30, "frames per second")
	engine := fs.String("engine", "voicevox", "engine type: voicevox or mock")
	engineURL := fs.String("engine-url", "http://aivisspeech:10101", "TTS engine URL")
	speaker := fs.Int("speaker", shared.DefaultSpeakerID, "speaker ID")
	speed := fs.Float64("speed", shared.DefaultSpeedScale, "speed scale")
	intonation := fs.Float64("intonation", shared.DefaultIntonationScale, "intonation scale")
	padMs := fs.Int("pad-ms", shared.DefaultPadMs, "padding silence in ms")

	output := fs.String("output", "", "output mp4 path (default: <out>/digest.mp4)")
	crf := fs.Int("crf", 20, "CRF quality")
	concurrency := fs.Int("concurrency", 0, "rendering concurrency")
	quiet := fs.Bool("quiet", false, "suppress progress output")

	description := fs.String("description", "", "content description")

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
		"-fps", fmt.Sprintf("%d", *fps),
		"-engine", *engine,
		"-engine-url", *engineURL,
		"-speaker", fmt.Sprintf("%d", *speaker),
		"-speed", fmt.Sprintf("%f", *speed),
		"-intonation", fmt.Sprintf("%f", *intonation),
		"-pad-ms", fmt.Sprintf("%d", *padMs),
	}
	if _, err := RunSynthesize(ctx, synthArgs); err != nil {
		return fmt.Errorf("all: synthesize step failed: %w", err)
	}

	// 2. Render
	renderArgs := []string{
		"-out", *outDir,
		"-crf", fmt.Sprintf("%d", *crf),
		"-concurrency", fmt.Sprintf("%d", *concurrency),
	}
	if *output != "" {
		renderArgs = append(renderArgs, "-output", *output)
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
	if *output != "" {
		publishArgs = append(publishArgs, "-output", *output)
	}
	if *description != "" {
		publishArgs = append(publishArgs, "-description", *description)
	}
	if err := RunPublish(ctx, publishArgs); err != nil {
		return fmt.Errorf("all: publish step failed: %w", err)
	}

	return nil
}

// Suppress unused imports check
var _ = filepath.Join
