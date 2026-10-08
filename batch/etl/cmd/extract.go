package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/k-wa-wa/pechka/batch/etl/shared"
)

type ExtractConfig struct {
	Device       string
	LocalMkvDir  string
	MinioBucket  string
	MinioURL     string
	MinioAccess  string
	MinioSecret  string
	MinioUseSSL  bool
	PechkaAPIURL string
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

func extractConfigFromEnv() ExtractConfig {
	return ExtractConfig{
		Device:       os.Getenv("DEVICE"),
		LocalMkvDir:  getenv("LOCAL_MKV_DIR", "/mnt/mkv"),
		MinioBucket:  mustGetenv("MINIO_BUCKET"),
		MinioURL:     mustGetenv("MINIO_URL"),
		MinioAccess:  mustGetenv("MINIO_ACCESS_KEY"),
		MinioSecret:  mustGetenv("MINIO_SECRET_KEY"),
		MinioUseSSL:  os.Getenv("MINIO_USE_SSL") == "true",
		PechkaAPIURL: os.Getenv("PECHKA_API_URL"),
	}
}

type IngestRequest struct {
	DiscLabel    string `json:"disc_label"`
	ContentTitle string `json:"content_title"`
}

func triggerIngestAPI(ctx context.Context, apiURL, discLabel, contentTitle string) error {
	if apiURL == "" {
		log.Println("PECHKA_API_URL is not set, skipping Ingest API trigger")
		return nil
	}

	urlStr := apiURL
	if strings.HasPrefix(urlStr, "http://") {
		urlStr = "https://" + strings.TrimPrefix(urlStr, "http://")
	} else if !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	urlStr = fmt.Sprintf("%s/api/v1/contents/ingest", strings.TrimSuffix(urlStr, "/"))

	if contentTitle == "" {
		contentTitle = fmt.Sprintf("Auto Ingested from Bluray Extractor VM (Label: %s)", discLabel)
	}

	reqBody := IngestRequest{
		DiscLabel:    discLabel,
		ContentTitle: contentTitle,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal ingest request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ingest API returned non-2xx status: %d", resp.StatusCode)
	}

	log.Printf("Successfully triggered Ingest API for disc: %s (url: %s)", discLabel, urlStr)
	return nil
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

var insecureTransport http.RoundTripper = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

func minioEndpoint(rawURL string, useSSLDefault bool) (string, bool) {
	if u, err := url.Parse(rawURL); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Host, u.Scheme == "https"
	}
	return rawURL, useSSLDefault
}

func uploadMKVToMinIO(ctx context.Context, cfg ExtractConfig, localDir, discLabel string) ([]MkvFile, error) {
	endpoint, secure := minioEndpoint(cfg.MinioURL, cfg.MinioUseSSL)
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure:    secure,
		Transport: insecureTransport,
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
			_, err = shared.FPutObjectWithRetry(ctx, minioClient, cfg.MinioBucket, objectKey, localPath, minio.PutObjectOptions{
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

func scanMinioMkvFiles(ctx context.Context, cfg ExtractConfig, discLabel string) ([]MkvFile, error) {
	endpoint, secure := minioEndpoint(cfg.MinioURL, cfg.MinioUseSSL)
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.MinioAccess, cfg.MinioSecret, ""),
		Secure:    secure,
		Transport: insecureTransport,
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

func ejectDisc(device string) {
	if device == "" {
		return
	}
	log.Printf("Ejecting disc on device: %s", device)
	cmd := exec.Command("eject", device)
	if err := cmd.Run(); err != nil {
		log.Printf("WARNING: failed to eject disc on %s: %v", device, err)
	} else {
		log.Printf("Successfully ejected disc on %s", device)
	}
}

func getDiscLabel(device string) (string, error) {
	cmd := exec.Command("blkid", "-o", "value", "-s", "LABEL", device)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	label := strings.TrimSpace(string(out))
	if label == "" {
		return "", fmt.Errorf("no label found for device %s", device)
	}
	return label, nil
}

func getDiscUUID(device string) (string, error) {
	cmd := exec.Command("blkid", "-o", "value", "-s", "UUID", device)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	uuid := strings.TrimSpace(string(out))
	if uuid == "" {
		return "", fmt.Errorf("no UUID found for device %s", device)
	}
	return uuid, nil
}

// resolveDiscKey returns a unique key for the disc.
// If label is "DVD_VIDEO_RECORDER" (standard for domestic BD recorders),
// we append the disc's filesystem UUID (or timestamp fallback) so that
// each recorded disc has its own isolated directory and database record.
func resolveDiscKey(device, label string) string {
	if strings.EqualFold(label, "DVD_VIDEO_RECORDER") {
		uuid, err := getDiscUUID(device)
		if err == nil && uuid != "" {
			return fmt.Sprintf("%s_%s", label, uuid)
		}
		log.Printf("WARNING: failed to read UUID for %s: %v. Using timestamp fallback.", device, err)
		return fmt.Sprintf("%s_%s", label, time.Now().Format("20060102150405"))
	}
	return label
}

func resolveDiscIndex(device string) (int, error) {
	cmd := exec.Command("makemkvcon", "-r", "info", "disc:9999")
	out, _ := cmd.Output()

	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`^DRV:(\d+),\d+,\d+,\d+,"[^"]*","[^"]*","([^"]*)"`)

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			idx, _ := strconv.Atoi(m[1])
			devPath := m[2]
			if devPath == device {
				return idx, nil
			}
		}
	}
	return 0, nil
}

func RunExtract(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	cfg := extractConfigFromEnv()

	node, err := snowflake.NewNode(1)
	if err != nil {
		return fmt.Errorf("failed to create snowflake node: %w", err)
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
			return fmt.Errorf("DEVICE env var is required in auto mode")
		}
		rawLabel, err := getDiscLabel(cfg.Device)
		if err != nil {
			log.Printf("No disc detected: %v. Skipping extraction.", err)
			writeOutputs("", []MkvFile{})
			return nil
		}
		discLabel = resolveDiscKey(cfg.Device, rawLabel)
	}

	var mkvFiles []MkvFile
	if manualLabel == "" {
		outputDir := filepath.Join(cfg.LocalMkvDir, discLabel)
		if err := os.RemoveAll(outputDir); err != nil {
			log.Printf("WARNING: failed to remove existing output directory: %v", err)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		discIndex, err := resolveDiscIndex(cfg.Device)
		if err != nil {
			return fmt.Errorf("failed to resolve MakeMKV disc index for device %s: %w", cfg.Device, err)
		}

		log.Printf("Extracting disc: %s (MakeMKV disc:%d)", discLabel, discIndex)
		cmd := exec.Command("makemkvcon", "mkv", fmt.Sprintf("disc:%d", discIndex), "all", outputDir)
		cmd.Stdin = strings.NewReader(strings.Repeat("y\n", 100))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("makemkvcon failed: %w", err)
		}

		log.Printf("Uploading extracted MKV files to MinIO...")
		mkvFiles, err = uploadMKVToMinIO(ctx, cfg, outputDir, discLabel)
		if err != nil {
			return fmt.Errorf("failed to upload MKV files to MinIO: %w", err)
		}

		log.Printf("Cleaning up local temporary directory: %s", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			log.Printf("WARNING: failed to cleanup local output directory: %v", err)
		}

		var mainTitle string
		if len(mkvFiles) > 0 {
			mainTitle = mkvFiles[0].Title
		}
		if err := triggerIngestAPI(ctx, cfg.PechkaAPIURL, discLabel, mainTitle); err != nil {
			log.Printf("WARNING: failed to trigger Ingest API: %v", err)
		}

		ejectDisc(cfg.Device)
	} else {
		log.Printf("Manual mode. Scanning MinIO MKV files...")
		mkvFiles, err = scanMinioMkvFiles(ctx, cfg, discLabel)
		if err != nil {
			return fmt.Errorf("failed to scan MinIO MKV files: %w", err)
		}
	}

	writeOutputs(discLabel, mkvFiles)
	log.Printf("Extraction/scan complete for label: %s. Found %d MKV files.", discLabel, len(mkvFiles))
	return nil
}
