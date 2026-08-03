package shared

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var contentTypes = map[string]string{
	".m3u8": "application/x-mpegURL",
	".ts":   "video/MP2T",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

func HLSPrefix(shortID string) string {
	return fmt.Sprintf("resources/hls/%s", shortID)
}

func ThumbnailKey(shortID string) string {
	return fmt.Sprintf("thumbnails/%s/thumb_01.jpg", shortID)
}

type Storage struct {
	Bucket string
	Client *s3.Client
}

func NewStorageFromEnv(ctx context.Context) (*Storage, error) {
	minioURL, err := getRequiredEnv("MINIO_URL")
	if err != nil {
		return nil, err
	}
	accessKey, err := getRequiredEnv("MINIO_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	secretKey, err := getRequiredEnv("MINIO_SECRET_KEY")
	if err != nil {
		return nil, err
	}
	bucket, err := getRequiredEnv("MINIO_BUCKET")
	if err != nil {
		return nil, err
	}

	useSSL := strings.ToLower(os.Getenv("MINIO_USE_SSL")) == "true"
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s", scheme, strings.TrimRight(minioURL, "/"))

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Storage{
		Bucket: bucket,
		Client: client,
	}, nil
}

func (s *Storage) PutFile(ctx context.Context, localPath, key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(localPath))
	cType, ok := contentTypes[ext]
	if !ok {
		cType = "application/octet-stream"
	}

	_, err = s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(cType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to s3://%s/%s: %w", localPath, s.Bucket, key, err)
	}
	return nil
}

func (s *Storage) PutText(ctx context.Context, bodyText, key string) error {
	ext := strings.ToLower(filepath.Ext(key))
	cType, ok := contentTypes[ext]
	if !ok {
		cType = "text/plain"
	}

	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(bodyText)),
		ContentType: aws.String(cType),
	})
	if err != nil {
		return fmt.Errorf("failed to put text to s3://%s/%s: %w", s.Bucket, key, err)
	}
	return nil
}

func (s *Storage) PutDir(ctx context.Context, localDir, prefix string) (int, error) {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read dir %s: %w", localDir, err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			localPath := filepath.Join(localDir, entry.Name())
			key := fmt.Sprintf("%s/%s", strings.TrimRight(prefix, "/"), entry.Name())
			if err := s.PutFile(ctx, localPath, key); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func getRequiredEnv(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf("%s env var is required", name)
	}
	return val, nil
}
