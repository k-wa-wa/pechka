package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const masterPlaylist = `#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS="avc1.640028,mp4a.40.2"
original.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=6192000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="Audio Only",DEFAULT=NO,URI="audio.m3u8"
`

type Config struct {
	ShortID     string
	ContentID   string
	OutputDir   string
	MinioBucket string
	MinioURL    string
	MinioAccess string
	MinioSecret string
	MinioUseSSL bool
	PostgresDSN string
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
		ShortID:     mustGetenv("SHORT_ID"),
		ContentID:   mustGetenv("CONTENT_ID"),
		OutputDir:   mustGetenv("OUTPUT_DIR"),
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

	masterPath := filepath.Join(cfg.OutputDir, "master.m3u8")
	if err := os.WriteFile(masterPath, []byte(masterPlaylist), 0644); err != nil {
		log.Fatalf("failed to write master playlist: %v", err)
	}

	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("failed to create MinIO client: %v", err)
	}

	objectKey := fmt.Sprintf("resources/hls/%s/master.m3u8", cfg.ShortID)
	_, err = minioClient.FPutObject(ctx, cfg.MinioBucket, objectKey, masterPath, minio.PutObjectOptions{
		ContentType: "application/vnd.apple.mpegurl",
	})
	if err != nil {
		log.Fatalf("failed to upload master playlist: %v", err)
	}

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	hlsKey := fmt.Sprintf("resources/hls/%s/master.m3u8", cfg.ShortID)
	_, err = tx.Exec(ctx,
		"INSERT INTO video_variants (content_id, variant_type, hls_key) VALUES ($1, 'master', $2)",
		cfg.ContentID, hlsKey,
	)
	if err != nil {
		log.Fatalf("failed to insert master variant: %v", err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE contents SET status = 'ready', published_at = NOW() WHERE id = $1",
		cfg.ContentID,
	)
	if err != nil {
		log.Fatalf("failed to mark content ready: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit transaction: %v", err)
	}

	log.Println("Master playlist generated and content marked as ready.")
}
