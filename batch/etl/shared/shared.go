package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type MinioConfig struct {
	Bucket    string
	URL       string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func MinioConfigFromEnv() MinioConfig {
	return MinioConfig{
		Bucket:    os.Getenv("MINIO_BUCKET"),
		URL:       os.Getenv("MINIO_URL"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	}
}

func GetPostgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
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
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

type VariantInfo struct {
	VariantType string
	Bandwidth   *int
	Resolution  *string
	Codecs      *string
}

func IntPtr(v int) *int     { return &v }
func StrPtr(v string) *string { return &v }

func BuildVariantInfo(variantType string) VariantInfo {
	variantMap := map[string]VariantInfo{
		"original": {VariantType: "original"},
		"1080p":    {VariantType: "1080p", Bandwidth: IntPtr(6192000), Resolution: StrPtr("1920x1080"), Codecs: StrPtr("avc1.640028,mp4a.40.2")},
		"720p":     {VariantType: "720p", Bandwidth: IntPtr(3128000), Resolution: StrPtr("1280x720"), Codecs: StrPtr("avc1.4d001f,mp4a.40.2")},
		"480p":     {VariantType: "480p", Bandwidth: IntPtr(1628000), Resolution: StrPtr("854x480"), Codecs: StrPtr("avc1.42e01f,mp4a.40.2")},
		"audio":    {VariantType: "audio", Bandwidth: IntPtr(192000), Codecs: StrPtr("mp4a.40.2")},
	}
	if vi, ok := variantMap[variantType]; ok {
		return vi
	}
	return VariantInfo{VariantType: variantType}
}

func RegisterVariant(ctx context.Context, db *pgxpool.Pool, contentID, shortID, variantType string, vSpec VariantInfo) error {
	hlsKey := fmt.Sprintf("resources/hls/%s/%s.m3u8", shortID, variantType)
	if variantType == "master" {
		hlsKey = fmt.Sprintf("resources/hls/%s/master.m3u8", shortID)
	}
	_, err := db.Exec(ctx,
		`INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (content_id, variant_type) 
		 DO UPDATE SET hls_key = EXCLUDED.hls_key, bandwidth = EXCLUDED.bandwidth, resolution = EXCLUDED.resolution, codecs = EXCLUDED.codecs`,
		contentID, variantType, hlsKey, vSpec.Bandwidth, vSpec.Resolution, vSpec.Codecs,
	)
	return err
}

func GetMinioVariants(ctx context.Context, client *minio.Client, bucket, shortID string) ([]VariantInfo, error) {
	prefix := fmt.Sprintf("resources/hls/%s/", shortID)
	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	variantMap := map[string]VariantInfo{
		"master.m3u8":   {VariantType: "master"},
		"original.m3u8": {VariantType: "original"},
		"1080p.m3u8":    {VariantType: "1080p", Bandwidth: IntPtr(6192000), Resolution: StrPtr("1920x1080"), Codecs: StrPtr("avc1.640028,mp4a.40.2")},
		"720p.m3u8":     {VariantType: "720p", Bandwidth: IntPtr(3128000), Resolution: StrPtr("1280x720"), Codecs: StrPtr("avc1.4d001f,mp4a.40.2")},
		"480p.m3u8":     {VariantType: "480p", Bandwidth: IntPtr(1628000), Resolution: StrPtr("854x480"), Codecs: StrPtr("avc1.42e01f,mp4a.40.2")},
		"audio.m3u8":    {VariantType: "audio", Bandwidth: IntPtr(192000), Codecs: StrPtr("mp4a.40.2")},
	}

	var found []VariantInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("list objects: %w", object.Err)
		}
		parts := strings.Split(object.Key, "/")
		filename := parts[len(parts)-1]

		if vi, ok := variantMap[filename]; ok {
			found = append(found, vi)
		}
	}
	return found, nil
}

func GenerateAndUploadMasterPlaylist(ctx context.Context, client *minio.Client, bucket, shortID string, variants []VariantInfo) (bool, error) {
	var hasVideo bool
	for _, v := range variants {
		if v.VariantType != "audio" && v.VariantType != "master" {
			hasVideo = true
			break
		}
	}
	if !hasVideo {
		log.Printf("No video variants found. master.m3u8 is not generated.")
		return false, nil
	}

	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n\n")

	order := []string{"original", "1080p", "720p", "480p"}
	for _, target := range order {
		var found *VariantInfo
		for _, v := range variants {
			if v.VariantType == target {
				found = &v
				break
			}
		}
		if found == nil {
			continue
		}

		if found.VariantType == "original" {
			sb.WriteString("#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS=\"avc1.640028,mp4a.40.2\"\n")
			sb.WriteString("original.m3u8\n\n")
		} else {
			bandwidth := 0
			if found.Bandwidth != nil {
				bandwidth = *found.Bandwidth
			}
			resolution := ""
			if found.Resolution != nil {
				resolution = *found.Resolution
			}
			codecs := "avc1.640028,mp4a.40.2"
			if found.Codecs != nil {
				codecs = *found.Codecs
			}
			sb.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,CODECS=\"%s\"\n", bandwidth, resolution, codecs))
			sb.WriteString(fmt.Sprintf("%s.m3u8\n\n", found.VariantType))
		}
	}

	hasAudio := false
	for _, v := range variants {
		if v.VariantType == "audio" {
			hasAudio = true
			break
		}
	}
	if hasAudio {
		sb.WriteString("#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"audio\",NAME=\"Audio Only\",DEFAULT=NO,URI=\"audio.m3u8\"\n")
	}

	content := sb.String()
	log.Printf("Generated master.m3u8 content:\n%s", content)

	objectKey := fmt.Sprintf("resources/hls/%s/master.m3u8", shortID)
	reader := strings.NewReader(content)
	_, err := client.PutObject(ctx, bucket, objectKey, reader, int64(len(content)), minio.PutObjectOptions{
		ContentType: "application/x-mpegURL",
	})
	if err != nil {
		return false, fmt.Errorf("upload master.m3u8: %w", err)
	}
	log.Printf("Successfully uploaded master.m3u8 to MinIO key: %s", objectKey)
	return true, nil
}
