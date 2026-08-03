package shared

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	ContentTypeContentType = "video"
	ShortIDMax             = 50
)

var defaultTags = []string{"tech-feed"}
var nonAlphaNumRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func ShortIDFor(sourceKey string) string {
	slug := strings.Trim(nonAlphaNumRegex.ReplaceAllString(sourceKey, "-"), "-")
	slug = strings.ToLower(slug)
	if slug == "" {
		slug = "tech-feed"
	}
	if len(slug) <= ShortIDMax {
		return slug
	}

	h := sha1.New()
	h.Write([]byte(sourceKey))
	digest := hex.EncodeToString(h.Sum(nil))[:8]

	truncated := strings.TrimRight(slug[:ShortIDMax-9], "-")
	return fmt.Sprintf("%s-%s", truncated, digest)
}

func ConnectCatalogDB(ctx context.Context) (*pgx.Conn, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			return nil, fmt.Errorf("POSTGRES_DSN or DB_HOST env var is required")
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		sslmode := os.Getenv("SSL_MODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return conn, nil
}

func UpsertContent(ctx context.Context, conn *pgx.Conn, sourceKey, title, description string, durationSeconds int, tags []string) (string, string, error) {
	shortID := ShortIDFor(sourceKey)
	if tags == nil {
		tags = defaultTags
	}

	query := `
	INSERT INTO contents
		(short_id, content_type, title, description, status, duration_seconds, tags)
	VALUES ($1, $2, $3, $4, 'processing', $5, $6)
	ON CONFLICT (short_id)
	DO UPDATE SET
		title = EXCLUDED.title,
		description = EXCLUDED.description,
		duration_seconds = EXCLUDED.duration_seconds,
		tags = EXCLUDED.tags,
		status = 'processing',
		updated_at = NOW()
	RETURNING id, short_id
	`

	var contentID string
	var retShortID string
	err := conn.QueryRow(ctx, query, shortID, ContentTypeContentType, title, description, durationSeconds, tags).Scan(&contentID, &retShortID)
	if err != nil {
		return "", "", fmt.Errorf("upsert_content failed: %w", err)
	}

	return contentID, retShortID, nil
}

func RegisterVariant(ctx context.Context, conn *pgx.Conn, contentID, shortID, variantType string, bandwidth *int, resolution, codecs *string) error {
	hlsKey := fmt.Sprintf("resources/hls/%s/%s.m3u8", shortID, variantType)
	query := `
	INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (content_id, variant_type)
	DO UPDATE SET hls_key = EXCLUDED.hls_key, bandwidth = EXCLUDED.bandwidth,
	              resolution = EXCLUDED.resolution, codecs = EXCLUDED.codecs
	`

	_, err := conn.Exec(ctx, query, contentID, variantType, hlsKey, bandwidth, resolution, codecs)
	if err != nil {
		return fmt.Errorf("register_variant failed for %s: %w", variantType, err)
	}
	return nil
}

func SetThumbnail(ctx context.Context, conn *pgx.Conn, contentID, key string) error {
	query := `UPDATE contents SET thumbnail_key = $1 WHERE id = $2`
	_, err := conn.Exec(ctx, query, key, contentID)
	if err != nil {
		return fmt.Errorf("set_thumbnail failed: %w", err)
	}
	return nil
}

func MarkReady(ctx context.Context, conn *pgx.Conn, contentID string) error {
	query := `UPDATE contents SET status = 'ready', published_at = COALESCE(published_at, NOW()) WHERE id = $1`
	_, err := conn.Exec(ctx, query, contentID)
	if err != nil {
		return fmt.Errorf("mark_ready failed: %w", err)
	}
	return nil
}
