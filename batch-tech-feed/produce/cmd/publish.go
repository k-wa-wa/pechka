package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

func RunPublish(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	outDir := fs.String("out", "", "working directory")
	output := fs.String("output", "", "mp4 to publish (default: <out>/digest.mp4)")
	description := fs.String("description", "", "content description")

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
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	mp4Path := *output
	if mp4Path == "" {
		mp4Path = filepath.Join(*outDir, "digest.mp4")
	}

	if _, err := os.Stat(mp4Path); err != nil {
		return fmt.Errorf("%s not found; run the 'build' or 'render' step first", mp4Path)
	}

	dateStr := manifest.DigestDate
	if dateStr == "" {
		dateStr = "unknown"
	}
	sk := fmt.Sprintf("tech-feed:%s", dateStr)

	title := manifest.Title
	if title == "" {
		title = "技術ダイジェスト"
	}
	durationSec := int(math.Round(float64(manifest.TotalMs) / 1000.0))

	store, err := shared.NewStorageFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("failed to init storage: %w", err)
	}

	conn, err := shared.ConnectCatalogDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer conn.Close(ctx)

	contentID, shortID, err := shared.UpsertContent(ctx, conn, sk, title, *description, durationSec, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert content: %w", err)
	}
	fmt.Printf("content: id=%s short_id=%s source_key=%s\n", contentID, shortID, sk)

	tmpWorkDir, err := os.MkdirTemp("", "techfeed-hls-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpWorkDir)

	fmt.Println("transcoding to HLS...")
	variants, err := shared.TranscodeHLS(mp4Path, tmpWorkDir)
	if err != nil {
		return fmt.Errorf("failed to transcode HLS: %w", err)
	}

	prefix := shared.HLSPrefix(shortID)
	fmt.Printf("uploading HLS to s3://%s/%s/ ...\n", store.Bucket, prefix)
	count, err := store.PutDir(ctx, tmpWorkDir, prefix)
	if err != nil {
		return fmt.Errorf("failed to upload HLS dir: %w", err)
	}

	if err := store.PutText(ctx, shared.MasterPlaylist(variants), fmt.Sprintf("%s/master.m3u8", prefix)); err != nil {
		return fmt.Errorf("failed to put master.m3u8: %w", err)
	}
	fmt.Printf("  %d file(s) + master.m3u8\n", count)

	thumbPath := filepath.Join(tmpWorkDir, "thumb_01.jpg")
	if _, err := shared.GenerateThumbnail(mp4Path, thumbPath, 2.0); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	thumbKey := shared.ThumbnailKey(shortID)
	if err := store.PutFile(ctx, thumbPath, thumbKey); err != nil {
		return fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	if err := shared.SetThumbnail(ctx, conn, contentID, thumbKey); err != nil {
		return fmt.Errorf("failed to set thumbnail in DB: %w", err)
	}
	fmt.Printf("  thumbnail -> %s\n", thumbKey)

	// register master variant
	if err := shared.RegisterVariant(ctx, conn, contentID, shortID, "master", nil, nil, nil); err != nil {
		return fmt.Errorf("failed to register master variant: %w", err)
	}

	for _, v := range variants {
		var bw *int
		var res *string
		var c *string
		if v.Bandwidth > 0 {
			val := v.Bandwidth
			bw = &val
		}
		if v.Resolution != "" {
			val := v.Resolution
			res = &val
		}
		if v.Codecs != "" {
			val := v.Codecs
			c = &val
		}
		if err := shared.RegisterVariant(ctx, conn, contentID, shortID, v.Name, bw, res, c); err != nil {
			return fmt.Errorf("failed to register variant %s: %w", v.Name, err)
		}
	}
	fmt.Printf("registered %d variant(s)\n", len(variants)+1)

	if err := shared.MarkReady(ctx, conn, contentID); err != nil {
		return fmt.Errorf("failed to mark content ready: %w", err)
	}
	fmt.Printf("content is ready: /contents/%s\n", shortID)

	return nil
}
