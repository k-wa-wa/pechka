package shared

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	upsertContentQuery = `
	INSERT INTO contents
		(short_id, content_type, channel_id, title, description, status, duration_seconds, tags)
	VALUES ($1, $2, $3, $4, $5, 'processing', $6, $7)
	ON CONFLICT (short_id)
	DO UPDATE SET
		channel_id = EXCLUDED.channel_id,
		title = EXCLUDED.title,
		description = EXCLUDED.description,
		duration_seconds = EXCLUDED.duration_seconds,
		tags = EXCLUDED.tags,
		status = 'processing',
		updated_at = NOW()
	WHERE contents.deleted_at IS NULL
	RETURNING id, short_id
	`

	registerVariantQuery = `
	WITH parent AS (
		SELECT id FROM contents WHERE id = $1 AND deleted_at IS NULL FOR SHARE
	)
	INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
	SELECT parent.id, $2, $3, $4, $5, $6 FROM parent
	ON CONFLICT (content_id, variant_type)
	DO UPDATE SET hls_key = EXCLUDED.hls_key, bandwidth = EXCLUDED.bandwidth,
	              resolution = EXCLUDED.resolution, codecs = EXCLUDED.codecs
	`

	setThumbnailQuery = `UPDATE contents SET thumbnail_key = $1 WHERE id = $2 AND deleted_at IS NULL`
	markReadyQuery    = `UPDATE contents SET status = 'ready', published_at = COALESCE(published_at, NOW()) WHERE id = $1 AND deleted_at IS NULL`
)

type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type dbExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

const (
	ContentTypeContentType = "video"
	DefaultChannelID       = "tech-feed"
	ShortIDMax             = 50
	RandomSuffixLen        = 6 // 6桁のランダムヘキサ(例: a1b2c3)
)

var defaultTags = []string{"tech-feed"}
var nonAlphaNumRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func randomHex(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())[:byteLen*2]
	}
	return hex.EncodeToString(b)
}

func ShortIDFor(sourceKey string) string {
	slug := strings.Trim(nonAlphaNumRegex.ReplaceAllString(sourceKey, "-"), "-")
	slug = strings.ToLower(slug)
	if slug == "" {
		slug = "tech-feed"
	}

	suffix := randomHex(RandomSuffixLen / 2) // 3 bytes -> 6 hex chars
	maxSlugLen := ShortIDMax - (len(suffix) + 1)

	if len(slug) > maxSlugLen {
		h := sha1.New()
		h.Write([]byte(sourceKey))
		digest := hex.EncodeToString(h.Sum(nil))[:6]
		slug = strings.TrimRight(slug[:maxSlugLen-7], "-") + "-" + digest
	}

	return fmt.Sprintf("%s-%s", slug, suffix)
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
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return conn, nil
}

func UpsertContent(ctx context.Context, conn *pgx.Conn, sourceKey, title, description string, durationSeconds int, tags []string) (string, string, error) {
	return UpsertContentWithQuerier(ctx, conn, sourceKey, title, description, durationSeconds, tags)
}

func UpsertContentWithQuerier(ctx context.Context, conn dbQuerier, sourceKey, title, description string, durationSeconds int, tags []string) (string, string, error) {
	shortID := ShortIDFor(sourceKey)
	if tags == nil {
		tags = defaultTags
	}
	channelID := strings.TrimSpace(os.Getenv("CHANNEL_ID"))
	if channelID == "" {
		channelID = DefaultChannelID
	}

	var contentID string
	var retShortID string
	err := conn.QueryRow(ctx, upsertContentQuery, shortID, ContentTypeContentType, channelID, title, description, durationSeconds, tags).Scan(&contentID, &retShortID)
	if err != nil {
		return "", "", fmt.Errorf("upsert_content failed: %w", err)
	}

	return contentID, retShortID, nil
}

func RegisterVariant(ctx context.Context, conn *pgx.Conn, contentID, shortID, variantType string, bandwidth *int, resolution, codecs *string) error {
	return RegisterVariantWithExec(ctx, conn, contentID, shortID, variantType, bandwidth, resolution, codecs)
}

func RegisterVariantWithExec(ctx context.Context, conn dbExecer, contentID, shortID, variantType string, bandwidth *int, resolution, codecs *string) error {
	hlsKey := fmt.Sprintf("resources/hls/%s/%s.m3u8", shortID, variantType)
	tag, err := conn.Exec(ctx, registerVariantQuery, contentID, variantType, hlsKey, bandwidth, resolution, codecs)
	if err != nil {
		return fmt.Errorf("register_variant failed for %s: %w", variantType, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("register_variant failed for %s: content %s not found or deleted", variantType, contentID)
	}
	return nil
}

func SetThumbnail(ctx context.Context, conn *pgx.Conn, contentID, key string) error {
	return SetThumbnailWithExec(ctx, conn, contentID, key)
}

func SetThumbnailWithExec(ctx context.Context, conn dbExecer, contentID, key string) error {
	tag, err := conn.Exec(ctx, setThumbnailQuery, key, contentID)
	if err != nil {
		return fmt.Errorf("set_thumbnail failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set_thumbnail failed: content %s not found or deleted", contentID)
	}
	return nil
}

func MarkReady(ctx context.Context, conn *pgx.Conn, contentID string) error {
	return MarkReadyWithExec(ctx, conn, contentID)
}

func MarkReadyWithExec(ctx context.Context, conn dbExecer, contentID string) error {
	tag, err := conn.Exec(ctx, markReadyQuery, contentID)
	if err != nil {
		return fmt.Errorf("mark_ready failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark_ready failed: content %s not found or deleted", contentID)
	}
	return nil
}
