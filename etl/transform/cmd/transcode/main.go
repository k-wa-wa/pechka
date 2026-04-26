package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	InputMKV    string
	OutputDir   string
	ShortID     string
	Variant     string
	ContentID   string
	MinioBucket string
	MinioURL    string
	MinioAccess string
	MinioSecret string
	MinioUseSSL bool
	PostgresDSN string
}

type variantMeta struct {
	bandwidth  *int
	resolution *string
	codecs     *string
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func configFromEnv() Config {
	return Config{
		InputMKV:    mustGetenv("INPUT_MKV"),
		OutputDir:   mustGetenv("OUTPUT_DIR"),
		ShortID:     mustGetenv("SHORT_ID"),
		Variant:     mustGetenv("VARIANT"),
		ContentID:   mustGetenv("CONTENT_ID"),
		MinioBucket: mustGetenv("MINIO_BUCKET"),
		MinioURL:    mustGetenv("MINIO_URL"),
		MinioAccess: mustGetenv("MINIO_ACCESS_KEY"),
		MinioSecret: mustGetenv("MINIO_SECRET_KEY"),
		MinioUseSSL: os.Getenv("MINIO_USE_SSL") == "true",
		PostgresDSN: mustGetenv("POSTGRES_DSN"),
	}
}

func main() {
	cfg := configFromEnv()
	ctx := context.Background()

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	log.Printf("Transcoding %s as variant=%s short_id=%s", cfg.InputMKV, cfg.Variant, cfg.ShortID)
	meta, err := transcode(cfg)
	if err != nil {
		log.Fatalf("transcode failed: %v", err)
	}

	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("failed to create MinIO client: %v", err)
	}

	log.Println("Uploading to MinIO...")
	if err := uploadVariant(ctx, minioClient, cfg); err != nil {
		log.Fatalf("failed to upload: %v", err)
	}

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	hlsKey := fmt.Sprintf("resources/hls/%s/%s.m3u8", cfg.ShortID, cfg.Variant)
	_, err = db.Exec(ctx,
		`INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		cfg.ContentID, cfg.Variant, hlsKey, meta.bandwidth, meta.resolution, meta.codecs,
	)
	if err != nil {
		log.Fatalf("failed to register variant: %v", err)
	}

	log.Printf("Variant %s registered successfully.", cfg.Variant)
}

func transcode(cfg Config) (variantMeta, error) {
	var args []string
	var meta variantMeta

	switch cfg.Variant {
	case "original":
		args = []string{
			"-i", cfg.InputMKV,
			"-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(cfg.OutputDir, "original_%04d.ts"),
			filepath.Join(cfg.OutputDir, "original.m3u8"),
		}
		meta = variantMeta{codecs: strPtr("avc1.640028,mp4a.40.2")}
	case "1080p":
		args = []string{
			"-i", cfg.InputMKV,
			"-vf", "scale=1920:1080", "-c:v", "libx264", "-preset", "fast",
			"-b:v", "6000k", "-maxrate", "6500k", "-bufsize", "12000k",
			"-c:a", "aac", "-b:a", "192k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(cfg.OutputDir, "1080p_%04d.ts"),
			filepath.Join(cfg.OutputDir, "1080p.m3u8"),
		}
		meta = variantMeta{bandwidth: intPtr(6192000), resolution: strPtr("1920x1080"), codecs: strPtr("avc1.640028,mp4a.40.2")}
	case "720p":
		args = []string{
			"-i", cfg.InputMKV,
			"-vf", "scale=1280:720", "-c:v", "libx264", "-preset", "fast",
			"-b:v", "3000k", "-maxrate", "3500k", "-bufsize", "6000k",
			"-c:a", "aac", "-b:a", "128k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(cfg.OutputDir, "720p_%04d.ts"),
			filepath.Join(cfg.OutputDir, "720p.m3u8"),
		}
		meta = variantMeta{bandwidth: intPtr(3128000), resolution: strPtr("1280x720"), codecs: strPtr("avc1.4d001f,mp4a.40.2")}
	case "480p":
		args = []string{
			"-i", cfg.InputMKV,
			"-vf", "scale=854:480", "-c:v", "libx264", "-preset", "fast",
			"-b:v", "1500k", "-maxrate", "2000k", "-bufsize", "3000k",
			"-c:a", "aac", "-b:a", "128k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(cfg.OutputDir, "480p_%04d.ts"),
			filepath.Join(cfg.OutputDir, "480p.m3u8"),
		}
		meta = variantMeta{bandwidth: intPtr(1628000), resolution: strPtr("854x480"), codecs: strPtr("avc1.42e01f,mp4a.40.2")}
	case "audio":
		args = []string{
			"-i", cfg.InputMKV,
			"-vn", "-c:a", "aac", "-b:a", "192k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(cfg.OutputDir, "audio_%04d.ts"),
			filepath.Join(cfg.OutputDir, "audio.m3u8"),
		}
		meta = variantMeta{bandwidth: intPtr(192000), codecs: strPtr("mp4a.40.2")}
	default:
		return variantMeta{}, fmt.Errorf("unknown variant: %s", cfg.Variant)
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return variantMeta{}, fmt.Errorf("ffmpeg: %w", err)
	}
	return meta, nil
}

func uploadVariant(ctx context.Context, client *minio.Client, cfg Config) error {
	entries, err := os.ReadDir(cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		localPath := filepath.Join(cfg.OutputDir, entry.Name())
		contentType := "application/octet-stream"
		if strings.HasSuffix(entry.Name(), ".m3u8") {
			contentType = "application/vnd.apple.mpegurl"
		} else if strings.HasSuffix(entry.Name(), ".ts") {
			contentType = "video/MP2T"
		}
		objectKey := fmt.Sprintf("resources/hls/%s/%s", cfg.ShortID, entry.Name())
		_, err := client.FPutObject(ctx, cfg.MinioBucket, objectKey, localPath, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("upload %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
