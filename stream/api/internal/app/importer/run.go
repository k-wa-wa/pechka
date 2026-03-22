package importer

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

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

type ImportJob struct {
	Path string
	Info os.FileInfo
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
	cdnBaseURL := strings.TrimRight(os.Getenv("CDN_BASE_URL"), "/")
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

	analyzerURL := os.Getenv("THUMBNAIL_ANALYZER_URL")
	if analyzerURL == "" {
		analyzerURL = "http://thumbnail-analyzer:8000"
	}
	scorer := NewHttpThumbnailScorer(analyzerURL)

	// Fetch nfs-admin group ID
	var nfsAdminGroupID uuid.UUID
	err = pgPool.QueryRow(ctx, "SELECT id FROM groups WHERE name = 'nfs-admin'").Scan(&nfsAdminGroupID)
	if err != nil {
		log.Printf("WARNING: nfs-admin group not found, using Nil UUID: %v", err)
		nfsAdminGroupID = uuid.Nil
	}

	log.Printf("INFO: Starting NFS Video Importer Batch")
	log.Printf("INFO: Scan Path: %s", nfsScanPath)

	maxWorkers := int(getEnvInt64("IMPORTER_MAX_WORKERS", 5))
	jobs := make(chan ImportJob, 100)
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				err := processVideo(ctx, job, nfsScanPath, repo, s3Client, minioBucket, cdnBaseURL, idGen, scorer, nfsAdminGroupID)
				if err != nil {
					log.Printf("ERROR[Worker-%d]: Failed to process %s: %v", workerID, job.Path, err)
				}
			}
		}(i)
	}

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

		jobs <- ImportJob{Path: path, Info: info}
		return nil
	})

	if err != nil {
		log.Fatalf("FAILED: Scan error: %v", err)
	}

	// Close the channel and wait for all workers
	close(jobs)
	wg.Wait()

	log.Println("INFO: Batch completed successfully.")
}

func processVideo(ctx context.Context, job ImportJob, nfsScanPath string, repo domain.ContentRepository, s3Client *s3.Client, minioBucket, cdnBaseURL string, idGen domain.ShortIDGenerator, scorer domain.ThumbnailScorer, nfsAdminGroupID uuid.UUID) error {
	path := job.Path
	info := job.Info

	relPath, err := filepath.Rel(nfsScanPath, path)
	if err != nil {
		return fmt.Errorf("filepath.Rel error: %w", err)
	}
	relDir := filepath.Dir(relPath)
	s3Prefix := fmt.Sprintf("videos/nfs/%s", relDir)
	masterS3Key := fmt.Sprintf("%s/%s", s3Prefix, filepath.Base(path))

	log.Printf("DEBUG: Found HLS master: %s (S3Key: %s)", relPath, masterS3Key)

	exists, err := repo.CheckDuplicateByS3Key(ctx, masterS3Key)
	if err != nil {
		return fmt.Errorf("CheckDuplicateByS3Key error: %w", err)
	}
	if exists {
		log.Printf("INFO: Skipping %s (already imported)", relPath)
		return nil
	}

	log.Printf("INFO: Importing %s...", relPath)

	dir := filepath.Dir(path)
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("os.ReadDir error: %w", err)
	}

	var eg errgroup.Group
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fName := f.Name()
		ext := filepath.Ext(fName)
		if ext != ".m3u8" && ext != ".ts" && ext != ".m4s" {
			continue
		}

		localFilePath := filepath.Join(dir, fName)
		s3Key := fmt.Sprintf("%s/%s", s3Prefix, fName)

		eg.Go(func() error {
			return uploadToS3(ctx, s3Client, localFilePath, minioBucket, s3Key)
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("S3 upload group error: %w", err)
	}

	videoID := uuid.New()
	var assets []domain.Asset
	assets = append(assets, domain.Asset{
		ID:        uuid.New(),
		AssetRole: domain.AssetRoleHLSMaster,
		S3Key:     masterS3Key,
		PublicURL: fmt.Sprintf("%s/%s", cdnBaseURL, masterS3Key),
	})

	durationSec := getVideoDuration(path)

	log.Printf("INFO: Generating thumbnail for %s (Duration: %.2fs)", filepath.Base(path), durationSec)
	bestThumbPath, err := generateThumbnailsWithScorer(ctx, path, videoID.String(), durationSec, scorer)
	if err == nil && bestThumbPath != "" {
		defer os.Remove(bestThumbPath)
		m3u8Base := strings.TrimSuffix(filepath.Base(path), ".m3u8")
		thumbnailKey := fmt.Sprintf("%s/%s_thumb.jpg", s3Prefix, m3u8Base)

		if err := uploadToS3(ctx, s3Client, bestThumbPath, minioBucket, thumbnailKey); err != nil {
			log.Printf("ERROR: Failed to upload thumbnail: %v", err)
		} else {
			assets = append(assets, domain.Asset{
				ID:        uuid.New(),
				AssetRole: domain.AssetRoleThumbnail,
				S3Key:     thumbnailKey,
				PublicURL: fmt.Sprintf("%s/%s", cdnBaseURL, thumbnailKey),
			})
		}
	} else {
		log.Printf("WARNING: Could not generate thumbnail for %s (using fallback): %v", path, err)
	}

	name := strings.TrimSuffix(info.Name(), ".m3u8")
	if name == "master" {
		name = filepath.Base(filepath.Dir(path))
	}

	now := time.Now()
	content := &domain.Content{
		ID:            videoID,
		ShortID:       idGen.Generate(),
		ContentType:   domain.ContentTypeVideo,
		Title:         name,
		Description:   "",
		Tags:          []string{},
		Visibility:    "private",
		AllowedGroups: []uuid.UUID{nfsAdminGroupID},
		PublishedAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
		VideoDetails: &domain.VideoDetails{
			Is360:           false,
			DurationSeconds: int(durationSec),
			Director:        "",
		},
		Assets: assets,
	}

	if err := repo.CreateContent(ctx, content); err != nil {
		return fmt.Errorf("DB registration error: %w", err)
	}

	log.Printf("SUCCESS: Imported %s as Content ID %s", relPath, videoID)
	return nil
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

func getVideoDuration(path string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err == nil {
		d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
		if err == nil {
			return d
		}
	}

	content, err := os.ReadFile(path)
	if err == nil {
		lines := strings.Split(string(content), "\n")
		var total float64
		for _, line := range lines {
			if strings.HasPrefix(line, "#EXTINF:") {
				valStr := strings.TrimPrefix(line, "#EXTINF:")
				valStr = strings.TrimSuffix(valStr, ",")
				v, _ := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
				total += v
			}
		}
		if total > 0 {
			return total
		}
	}
	return 0.0
}

func generateThumbnailsWithScorer(ctx context.Context, videoPath, videoID string, duration float64, scorer domain.ThumbnailScorer) (string, error) {
	points := []float64{0.0, 0.3 * duration, 0.6 * duration, 0.9 * duration}

	var bestTS float64 = 0.3 * duration // Default fallback to 30%
	res, err := scorer.Analyze(ctx, videoPath, points)
	if err != nil {
		log.Printf("WARNING: Scorer failed, using fallback 30%%: %v", err)
	} else {
		bestTS = res.BestTimestamp
	}

	thumbPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_thumb_best.jpg", videoID))
	cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%.3f", bestTS), "-i", videoPath, "-vframes", "1", "-q:v", "2", thumbPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg extract error at %.3f: %v, out: %s", bestTS, err, string(output))
	}

	return thumbPath, nil
}

