package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Device      string
	LocalMkvDir string
	MinioBucket string
	MinioURL    string
	MinioAccess string
	MinioSecret string
	MinioUseSSL bool
}

type MkvFile struct {
	MkvPath string `json:"mkv-path"`
	Label   string `json:"label"`
	Title   string `json:"title"`
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

func configFromEnv() Config {
	return Config{
		Device:      os.Getenv("DEVICE"),
		LocalMkvDir: getenv("LOCAL_MKV_DIR", "/mnt/mkv"),
		MinioBucket: mustGetenv("MINIO_BUCKET"),
		MinioURL:    mustGetenv("MINIO_URL"),
		MinioAccess: mustGetenv("MINIO_ACCESS_KEY"),
		MinioSecret: mustGetenv("MINIO_SECRET_KEY"),
		MinioUseSSL: os.Getenv("MINIO_USE_SSL") == "true",
	}
}

func writeOutputs(label string, mkvFiles []MkvFile) {
	if err := os.WriteFile("/tmp/bluray-label", []byte(label), 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/bluray-label: %v", err)
	}
	data, err := json.Marshal(mkvFiles)
	if err != nil {
		log.Printf("WARNING: failed to marshal mkv files: %v", err)
		data = []byte("[]")
	}
	if err := os.WriteFile("/tmp/mkv-files.json", data, 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/mkv-files.json: %v", err)
	}
}
func uploadMKVToMinIO(ctx context.Context, cfg Config, localDir, discLabel string) ([]MkvFile, error) {
	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	files, err := os.ReadDir(localDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read local directory: %w", err)
	}

	var uploadedFiles []MkvFile
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
			localPath := filepath.Join(localDir, f.Name())
			objectKey := fmt.Sprintf("mkv/%s/%s", discLabel, f.Name())

			log.Printf("Uploading %s to MinIO bucket %s as %s...", f.Name(), cfg.MinioBucket, objectKey)
			_, err = minioClient.FPutObject(ctx, cfg.MinioBucket, objectKey, localPath, minio.PutObjectOptions{
				ContentType: "video/x-matroska",
			})
			if err != nil {
				return nil, fmt.Errorf("failed to upload %s to MinIO: %w", f.Name(), err)
			}

			name := strings.TrimSuffix(f.Name(), ".mkv")
			uploadedFiles = append(uploadedFiles, MkvFile{
				MkvPath: objectKey,
				Label:   discLabel,
				Title:   name,
			})
		}
	}
	return uploadedFiles, nil
}

func scanMinioMkvFiles(ctx context.Context, cfg Config, discLabel string) ([]MkvFile, error) {
	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	prefix := fmt.Sprintf("mkv/%s/", discLabel)
	objectCh := minioClient.ListObjects(ctx, cfg.MinioBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var res []MkvFile
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects from MinIO: %w", object.Err)
		}
		if strings.HasSuffix(object.Key, ".mkv") {
			parts := strings.Split(object.Key, "/")
			filename := parts[len(parts)-1]
			name := strings.TrimSuffix(filename, ".mkv")

			res = append(res, MkvFile{
				MkvPath: object.Key,
				Label:   discLabel,
				Title:   name,
			})
		}
	}
	return res, nil
}

func main() {
	cfg := configFromEnv()
	ctx := context.Background()

	// Generate Snowflake ID beforehand
	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("failed to create snowflake node: %v", err)
	}
	shortID := node.Generate().String()
	if err := os.WriteFile("/tmp/short-id", []byte(shortID), 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/short-id: %v", err)
	}
	log.Printf("Generated shortID: %s", shortID)

	var discLabel string
	manualLabel := os.Getenv("DISC_LABEL")

	if manualLabel != "" {
		log.Printf("Manual mode. Using label: %s", manualLabel)
		discLabel = manualLabel
	} else {
		log.Printf("Auto mode. Detecting disc on device: %s", cfg.Device)
		if cfg.Device == "" {
			log.Fatal("DEVICE env var is required in auto mode")
		}
		var err error
		discLabel, err = getDiscLabel(cfg.Device)
		if err != nil {
			log.Printf("No disc detected: %v. Skipping extraction.", err)
			writeOutputs("", []MkvFile{})
			os.Exit(0)
		}
	}

	var mkvFiles []MkvFile
	if manualLabel == "" {
		outputDir := filepath.Join(cfg.LocalMkvDir, discLabel)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("failed to create output directory: %v", err)
		}

		log.Printf("Extracting disc: %s", discLabel)
		cmd := exec.Command("makemkvcon", "mkv", "disc:0", "all", outputDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("makemkvcon failed: %v", err)
		}

		log.Printf("Uploading extracted MKV files to MinIO...")
		mkvFiles, err = uploadMKVToMinIO(ctx, cfg, outputDir, discLabel)
		if err != nil {
			log.Fatalf("failed to upload MKV files to MinIO: %v", err)
		}

		log.Printf("Cleaning up local temporary directory: %s", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			log.Printf("WARNING: failed to cleanup local output directory: %v", err)
		}
	} else {
		log.Printf("Scanning existing files in MinIO for label: %s", discLabel)
		mkvFiles, err = scanMinioMkvFiles(ctx, cfg, discLabel)
		if err != nil {
			log.Fatalf("failed to scan MinIO MKV files: %v", err)
		}
	}

	log.Printf("Found %d MKV files", len(mkvFiles))

	writeOutputs(discLabel, mkvFiles)
}


func getDiscLabel(device string) (string, error) {
	out, err := exec.Command("blkid", device).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run blkid: %w, output: %s", err, string(out))
	}
	for _, part := range strings.Fields(string(out)) {
		if strings.HasPrefix(part, "LABEL=") {
			return strings.Trim(strings.TrimPrefix(part, "LABEL="), "\""), nil
		}
	}
	return "", fmt.Errorf("LABEL not found in blkid output")
}
