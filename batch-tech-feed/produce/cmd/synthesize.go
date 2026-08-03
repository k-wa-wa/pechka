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

func RunSynthesize(ctx context.Context, osArgs []string) (*shared.Manifest, error) {
	fs := flag.NewFlagSet("synthesize", flag.ContinueOnError)
	scriptPath := fs.String("script", "", "path to script.json")
	outDir := fs.String("out", "", "working directory for intermediate artifacts")
	fps := fs.Int("fps", 30, "frames per second")
	engine := fs.String("engine", "voicevox", "engine type: voicevox or mock")
	engineURL := fs.String("engine-url", "http://aivisspeech:10101", "TTS engine URL")
	speaker := fs.Int("speaker", shared.DefaultSpeakerID, "speaker ID")
	speed := fs.Float64("speed", shared.DefaultSpeedScale, "speed scale")
	intonation := fs.Float64("intonation", shared.DefaultIntonationScale, "intonation scale")
	padMs := fs.Int("pad-ms", shared.DefaultPadMs, "padding silence in ms")

	if err := fs.Parse(osArgs); err != nil {
		return nil, err
	}

	if *scriptPath == "" || *outDir == "" {
		return nil, fmt.Errorf("--script and --out parameters are required")
	}

	raw, err := os.ReadFile(*scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	var script shared.Script
	if err := json.Unmarshal(raw, &script); err != nil {
		return nil, fmt.Errorf("failed to parse script JSON: %w", err)
	}

	entries := shared.BuildTimeline(&script)
	fmt.Printf("script: %d section(s), %d line(s)\n", len(script.Sections), len(entries))

	synth, err := shared.BuildSynthesizer(*engine, *engineURL, *speaker, *speed, *intonation)
	if err != nil {
		return nil, err
	}

	fmt.Printf("synthesizing with engine=%s...\n", *engine)
	durations, err := shared.RunSynthesize(entries, *outDir, synth, *padMs)
	if err != nil {
		return nil, fmt.Errorf("synthesize failed: %w", err)
	}

	totalMs, err := shared.ApplyDurations(entries, durations)
	if err != nil {
		return nil, err
	}
	fmt.Printf("total narration: %.1fs\n", float64(totalMs)/1000.0)

	narrationPath, err := shared.BuildNarrationConcat(entries, *outDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build narration concat: %w", err)
	}

	manifest := shared.ToManifest(&script, entries, *fps, narrationPath)
	manifestPath := filepath.Join(*outDir, "manifest.json")
	if err := shared.SaveManifest(manifest, manifestPath); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Printf("manifest: %s (total %.1fs)\n", manifestPath, float64(manifest.TotalMs)/1000.0)
	return manifest, nil
}
