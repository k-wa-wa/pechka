package importer

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/infrastructure/idgen"
	"pechka/streaming-service/api/internal/infrastructure/postgres"
)

func getEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	var i int64
	fmt.Sscanf(val, "%d", &i)
	return i
}

func Run() {
	_ = godotenv.Load()

	ctx := context.Background()

	pgURL := os.Getenv("DATABASE_URL")
	pgPool, err := pgxpool.New(ctx, pgURL)
	if err != nil {
		log.Fatalf("FAILED: pg connect: %v", err)
	}
	defer pgPool.Close()

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	minioBucket := os.Getenv("MINIO_BUCKET")
	cdnBaseURL := os.Getenv("CDN_BASE_URL")
	nfsScanPath := os.Getenv("NFS_SCAN_PATH")

	if pgURL == "" || minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" || minioBucket == "" || nfsScanPath == "" {
		log.Fatalf("FAILED: Required environment variables are missing (DATABASE_URL, MINIO_*, NFS_SCAN_PATH)")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               minioEndpoint,
			HostnameImmutable: true,
			SigningRegion:     "us-east-1",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(minioAccessKey, minioSecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		log.Fatalf("FAILED: s3 config load: %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	repo := postgres.NewContentRepository(pgPool)
	nodeID := getEnvInt64("NODE_ID", 1)
	idGen := idgen.NewSnowflakeGenerator(nodeID)

	log.Printf("INFO: Starting NFS Video Importer Batch")
	log.Printf("INFO: Scan Path: %s", nfsScanPath)

	err = filepath.Walk(nfsScanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".m3u8" {
			return nil
		}

		relPath, _ := filepath.Rel(nfsScanPath, path)
		relDir := filepath.Dir(relPath)
		// Use a stable S3 prefix based on the NFS relative directory to allow idempotency check via S3 key
		s3Prefix := fmt.Sprintf("videos/nfs/%s", relDir)
		masterS3Key := fmt.Sprintf("%s/%s", s3Prefix, filepath.Base(path))

		log.Printf("DEBUG: Found HLS master: %s (S3Key: %s)", relPath, masterS3Key)

		exists, err := repo.CheckDuplicateByS3Key(ctx, masterS3Key)
		if err != nil {
			log.Printf("ERROR: Failed to check duplicate for %s: %v", masterS3Key, err)
			return nil
		}
		if exists {
			log.Printf("INFO: Skipping %s (already imported)", relPath)
			return nil
		}

		log.Printf("INFO: Importing %s...", relPath)

		dir := filepath.Dir(path)
		files, err := os.ReadDir(dir)
		if err != nil {
			log.Printf("ERROR: Failed to read directory %s: %v", dir, err)
			return nil
		}

		var assets []domain.Asset
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			fName := f.Name()
			ext := filepath.Ext(fName)
			if ext != ".m3u8" && ext != ".ts" {
				continue
			}

			localFilePath := filepath.Join(dir, fName)
			s3Key := fmt.Sprintf("%s/%s", s3Prefix, fName)

			if err := uploadToS3(ctx, s3Client, localFilePath, minioBucket, s3Key); err != nil {
				log.Printf("ERROR: Failed to upload %s: %v", fName, err)
				return nil
			}

			if ext == ".m3u8" {
				assets = append(assets, domain.Asset{
					ID:        uuid.New(),
					AssetRole: domain.AssetRoleHLSMaster,
					S3Key:     s3Key,
					PublicURL:  fmt.Sprintf("%s/%s", strings.TrimSuffix(cdnBaseURL, "/"), s3Key),
				})
			}
		}

		thumbnailKey := fmt.Sprintf("%s/thumbnail.jpg", s3Prefix)
		videoID := uuid.New() // Internal reference for temp file
		thumbPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_thumb.jpg", videoID.String()))
		
		var firstTS string
		for _, f := range files {
			fName := f.Name()
			if filepath.Ext(fName) == ".ts" && !strings.HasPrefix(fName, ".") {
				firstTS = filepath.Join(dir, fName)
				break
			}
		}

		if firstTS != "" {
			log.Printf("INFO: Generating thumbnail from %s...", filepath.Base(firstTS))
			cmd := exec.Command("ffmpeg", "-y", "-i", firstTS, "-ss", "00:00:00", "-vframes", "1", "-q:v", "2", thumbPath)
			if err := cmd.Run(); err != nil {
				log.Printf("WARNING: Failed to generate thumbnail: %v", err)
			} else {
				defer os.Remove(thumbPath)
				if err := uploadToS3(ctx, s3Client, thumbPath, minioBucket, thumbnailKey); err != nil {
					log.Printf("ERROR: Failed to upload thumbnail: %v", err)
				} else {
					assets = append(assets, domain.Asset{
						ID:        uuid.New(),
						AssetRole: domain.AssetRoleThumbnail,
						S3Key:     thumbnailKey,
						PublicURL:  fmt.Sprintf("%s/%s", strings.TrimSuffix(cdnBaseURL, "/"), thumbnailKey),
					})
				}
			}
		}

		name := strings.TrimSuffix(info.Name(), ".m3u8")
		if name == "master" {
			name = filepath.Base(filepath.Dir(path))
		}
		
		now := time.Now()
		video := &domain.Video{
			ID:              videoID,
			ShortID:         idGen.Generate(),
			Title:           name,
			Description:     "",
			Is360:           false,
			DurationSeconds: 0,
			Director:        "",
			CreatedAt:       now,
			UpdatedAt:       now,
			Assets:          assets,
		}

		if err := repo.CreateVideo(ctx, video); err != nil {
			log.Printf("ERROR: Failed to register %s to DB: %v", relPath, err)
			return nil
		}

		log.Printf("SUCCESS: Imported %s as Video ID %s", relPath, videoID)
		return nil
	})

	if err != nil {
		log.Fatalf("FAILED: Scan error: %v", err)
	}

	log.Println("INFO: Batch completed successfully.")
}

func uploadToS3(ctx context.Context, client *s3.Client, localPath, bucket, key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
	})
	return err
}
