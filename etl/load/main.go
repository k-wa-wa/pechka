package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	HLSResourceDir string
	MinioBucket    string
	MinioURL       string
	MinioAccess    string
	MinioSecret    string
	MinioUseSSL    bool
	PostgresDSN    string
	DiscLabel      string
	ContentTitle   string
	ContentType    string
	Is360          bool
}

func configFromEnv() Config {
	return Config{
		HLSResourceDir: hlsResourceDir(),
		MinioBucket:    mustGetenv("MINIO_BUCKET"),
		MinioURL:       mustGetenv("MINIO_URL"),
		MinioAccess:    mustGetenv("MINIO_ACCESS_KEY"),
		MinioSecret:    mustGetenv("MINIO_SECRET_KEY"),
		MinioUseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
		PostgresDSN:    postgresDSN(),
		DiscLabel:      mustGetenv("DISC_LABEL"),
		ContentTitle:   mustGetenv("CONTENT_TITLE"),
		ContentType:    getenv("CONTENT_TYPE", "video"),
		Is360:          os.Getenv("IS_360") == "true",
	}
}

// hlsResourceDir resolves HLS directory from environment.
// Prefers HLS_RESOURCE_DIR (nuage-cluster convention) over NFS_HLS_DIR.
func hlsResourceDir() string {
	if v := os.Getenv("HLS_RESOURCE_DIR"); v != "" {
		return v
	}
	return getenv("NFS_HLS_DIR", "/mnt/hls")
}

// postgresDSN builds a connection string from individual env vars (nuage-cluster convention)
// or falls back to POSTGRES_DSN if set.
func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := mustGetenv("DB_HOST")
	port := getenv("DB_PORT", "5432")
	user := mustGetenv("DB_USER")
	password := mustGetenv("DB_PASSWORD")
	dbname := mustGetenv("DB_NAME")
	sslmode := getenv("SSL_MODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := configFromEnv()
	ctx := context.Background()

	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("failed to create snowflake node: %v", err)
	}
	shortID := node.Generate().String()

	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("failed to create MinIO client: %v", err)
	}

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	discID, err := ensureDisc(ctx, db, cfg.DiscLabel)
	if err != nil {
		log.Fatalf("failed to ensure disc: %v", err)
	}

	contentID, err := insertContent(ctx, db, shortID, discID, cfg)
	if err != nil {
		log.Fatalf("failed to insert content: %v", err)
	}
	log.Printf("Content created: id=%s short_id=%s", contentID, shortID)

	hlsDir := filepath.Join(cfg.HLSResourceDir, cfg.DiscLabel)
	variants, err := uploadHLS(ctx, minioClient, cfg.MinioBucket, shortID, hlsDir)
	if err != nil {
		log.Fatalf("failed to upload HLS: %v", err)
	}
	log.Printf("Uploaded %d HLS variants", len(variants))

	if err := registerVariants(ctx, db, contentID, shortID, variants); err != nil {
		log.Fatalf("failed to register variants: %v", err)
	}

	if err := markContentReady(ctx, db, contentID); err != nil {
		log.Fatalf("failed to mark content ready: %v", err)
	}

	log.Printf("Load complete: short_id=%s", shortID)
}

func ensureDisc(ctx context.Context, db *pgxpool.Pool, label string) (string, error) {
	var id string
	err := db.QueryRow(ctx,
		"INSERT INTO discs (label) VALUES ($1) ON CONFLICT (label) DO UPDATE SET label = EXCLUDED.label RETURNING id",
		label,
	).Scan(&id)
	return id, err
}

func insertContent(ctx context.Context, db *pgxpool.Pool, shortID, discID string, cfg Config) (string, error) {
	var contentID string
	err := db.QueryRow(ctx,
		`INSERT INTO contents (short_id, content_type, disc_id, title, status, is_360)
		 VALUES ($1, $2, $3, $4, 'processing', $5)
		 RETURNING id`,
		shortID, cfg.ContentType, discID, cfg.ContentTitle, cfg.Is360,
	).Scan(&contentID)
	return contentID, err
}

type variantInfo struct {
	variantType string
	bandwidth   *int
	resolution  *string
	codecs      *string
}

func uploadHLS(ctx context.Context, client *minio.Client, bucket, shortID, hlsDir string) ([]variantInfo, error) {
	entries, err := os.ReadDir(hlsDir)
	if err != nil {
		return nil, fmt.Errorf("read hls dir %s: %w", hlsDir, err)
	}

	variantMap := map[string]variantInfo{
		"master.m3u8":   {variantType: "master"},
		"original.m3u8": {variantType: "original"},
		"1080p.m3u8":    {variantType: "1080p", bandwidth: intPtr(6192000), resolution: strPtr("1920x1080"), codecs: strPtr("avc1.640028,mp4a.40.2")},
		"720p.m3u8":     {variantType: "720p", bandwidth: intPtr(3128000), resolution: strPtr("1280x720"), codecs: strPtr("avc1.4d001f,mp4a.40.2")},
		"480p.m3u8":     {variantType: "480p", bandwidth: intPtr(1628000), resolution: strPtr("854x480"), codecs: strPtr("avc1.42e01f,mp4a.40.2")},
		"audio.m3u8":    {variantType: "audio", bandwidth: intPtr(192000), codecs: strPtr("mp4a.40.2")},
	}

	var uploaded []variantInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		localPath := filepath.Join(hlsDir, entry.Name())
		contentType := "application/octet-stream"
		if strings.HasSuffix(entry.Name(), ".m3u8") {
			contentType = "application/vnd.apple.mpegurl"
		} else if strings.HasSuffix(entry.Name(), ".ts") {
			contentType = "video/MP2T"
		}
		objectKey := fmt.Sprintf("resources/hls/%s/%s", shortID, entry.Name())
		_, err := client.FPutObject(ctx, bucket, objectKey, localPath, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return nil, fmt.Errorf("upload %s: %w", entry.Name(), err)
		}
		if vi, ok := variantMap[entry.Name()]; ok {
			uploaded = append(uploaded, vi)
		}
	}
	return uploaded, nil
}

func registerVariants(ctx context.Context, db *pgxpool.Pool, contentID, shortID string, variants []variantInfo) error {
	for _, v := range variants {
		hlsKey := fmt.Sprintf("resources/hls/%s/%s.m3u8", shortID, v.variantType)
		if v.variantType == "master" {
			hlsKey = fmt.Sprintf("resources/hls/%s/master.m3u8", shortID)
		}
		_, err := db.Exec(ctx,
			`INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			contentID, v.variantType, hlsKey, v.bandwidth, v.resolution, v.codecs,
		)
		if err != nil {
			return fmt.Errorf("insert variant %s: %w", v.variantType, err)
		}
	}
	return nil
}

func markContentReady(ctx context.Context, db *pgxpool.Pool, contentID string) error {
	_, err := db.Exec(ctx,
		"UPDATE contents SET status = 'ready', published_at = NOW() WHERE id = $1",
		contentID,
	)
	return err
}

func intPtr(v int) *int     { return &v }
func strPtr(v string) *string { return &v }
