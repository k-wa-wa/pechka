package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	latestPlaylistKey = "resources/hls/latest.m3u8"
	// latestPlaylistLockKey は api 側の pgRepo.LatestPlaylistLockKey と共有する 64-bit アドバイザリロックキー。
	latestPlaylistLockKey int64 = 0x706563686b610001
	latestPlaylistLimit         = 50
	latestPlaylistTimeout       = 15 * time.Second
	latestPlaylistQuery         = `SELECT short_id, title, published_at
		 FROM contents
		 WHERE status = 'ready' AND archived_at IS NULL AND deleted_at IS NULL
		 ORDER BY published_at DESC NULLS LAST, updated_at DESC, short_id ASC
		 LIMIT $1`
)

type contentRow struct {
	shortID     string
	title       string
	publishedAt *time.Time
}

type playlistRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type playlistTx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (playlistRows, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type playlistTxBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (playlistTx, error)
}

type pgxPoolBeginner struct {
	pool *pgxpool.Pool
}

func (b pgxPoolBeginner) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (playlistTx, error) {
	tx, err := b.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, err
	}
	return pgxPlaylistTx{tx: tx}, nil
}

type pgxPlaylistTx struct {
	tx pgx.Tx
}

func (t pgxPlaylistTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, arguments...)
}

func (t pgxPlaylistTx) Query(ctx context.Context, sql string, args ...any) (playlistRows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t pgxPlaylistTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxPlaylistTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type objectUploader interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
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

func main() {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, postgresDSN())
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	minioURL := mustGetenv("MINIO_URL")
	minioAccess := mustGetenv("MINIO_ACCESS_KEY")
	minioSecret := mustGetenv("MINIO_SECRET_KEY")
	minioBucket := mustGetenv("MINIO_BUCKET")
	minioSSL := os.Getenv("MINIO_USE_SSL") == "true"

	minioClient, err := minio.New(minioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccess, minioSecret, ""),
		Secure: minioSSL,
	})
	if err != nil {
		log.Fatalf("failed to create MinIO client: %v", err)
	}

	count, err := refreshPlaylist(ctx, pgxPoolBeginner{pool: db}, minioClient, minioBucket)
	if err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("Latest playlist updated: %d contents → s3://%s/%s", count, minioBucket, latestPlaylistKey)
}

func refreshPlaylist(ctx context.Context, beginner playlistTxBeginner, uploader objectUploader, bucket string) (int, error) {
	return refreshPlaylistWithTimeout(ctx, beginner, uploader, bucket, latestPlaylistTimeout)
}

func refreshPlaylistWithTimeout(ctx context.Context, beginner playlistTxBeginner, uploader objectUploader, bucket string, timeout time.Duration) (int, error) {
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx, err := beginner.BeginTx(opCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		// opCtx が期限切れでも接続とアドバイザリロックを確実に解放する
		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer rollbackCancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	if _, err := tx.Exec(opCtx, "SELECT pg_advisory_xact_lock($1)", latestPlaylistLockKey); err != nil {
		return 0, fmt.Errorf("failed to acquire playlist advisory lock: %w", err)
	}

	rows, err := tx.Query(opCtx, latestPlaylistQuery, latestPlaylistLimit)
	if err != nil {
		return 0, fmt.Errorf("failed to query contents: %w", err)
	}

	var contents []contentRow
	for rows.Next() {
		var c contentRow
		if err := rows.Scan(&c.shortID, &c.title, &c.publishedAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}
		contents = append(contents, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("row iteration error: %w", err)
	}
	rows.Close()

	playlist := buildPlaylist(contents)

	_, err = uploader.PutObject(opCtx, bucket, latestPlaylistKey,
		bytes.NewReader([]byte(playlist)),
		int64(len(playlist)),
		minio.PutObjectOptions{ContentType: "application/vnd.apple.mpegurl"},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to upload latest playlist: %w", err)
	}

	if err := tx.Commit(opCtx); err != nil {
		return 0, fmt.Errorf("failed to commit playlist refresh transaction: %w", err)
	}

	return len(contents), nil
}

func buildPlaylist(contents []contentRow) string {
	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	for _, c := range contents {
		buf.WriteString(fmt.Sprintf("#EXTINF:-1,%s\n", c.title))
		buf.WriteString(fmt.Sprintf("resources/hls/%s/master.m3u8\n", c.shortID))
	}
	return buf.String()
}
