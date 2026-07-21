package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const latestPlaylistKey = "resources/hls/latest.m3u8"

type contentRow struct {
	shortID     string
	title       string
	publishedAt time.Time
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

	rows, err := db.Query(ctx,
		`SELECT short_id, title, published_at
		 FROM contents
		 WHERE status = 'ready'
		 ORDER BY published_at DESC
		 LIMIT 50`,
	)
	if err != nil {
		log.Fatalf("failed to query contents: %v", err)
	}
	defer rows.Close()

	var contents []contentRow
	for rows.Next() {
		var c contentRow
		if err := rows.Scan(&c.shortID, &c.title, &c.publishedAt); err != nil {
			log.Fatalf("failed to scan row: %v", err)
		}
		contents = append(contents, c)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("row iteration error: %v", err)
	}

	playlist := buildPlaylist(contents)

	_, err = minioClient.PutObject(ctx, minioBucket, latestPlaylistKey,
		bytes.NewReader([]byte(playlist)),
		int64(len(playlist)),
		minio.PutObjectOptions{ContentType: "application/vnd.apple.mpegurl"},
	)
	if err != nil {
		log.Fatalf("failed to upload latest playlist: %v", err)
	}

	log.Printf("Latest playlist updated: %d contents → s3://%s/%s", len(contents), minioBucket, latestPlaylistKey)
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
