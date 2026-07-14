package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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

	"github.com/bwmarrin/snowflake"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
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

func configFromEnv() Config {
	return Config{
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

func triggerIngestAPI(ctx context.Context, apiURL, discLabel string) error {
	if apiURL == "" {
		log.Println("PECHKA_API_URL is not set, skipping Ingest API trigger")
		return nil
	}

	url := apiURL
	if strings.HasPrefix(url, "http://") {
		url = "https://" + strings.TrimPrefix(url, "http://")
	} else if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	url = fmt.Sprintf("%s/api/v1/contents/ingest", strings.TrimSuffix(url, "/"))

	reqBody := IngestRequest{
		DiscLabel:    discLabel,
		ContentTitle: fmt.Sprintf("Auto Ingested from Bluray Extractor VM (Label: %s)", discLabel),
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal ingest request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
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

	log.Printf("Successfully triggered Ingest API for disc: %s (url: %s)", discLabel, url)
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
// Cluster-internal services (e.g. *.cluster.wpc) use self-signed certs, so
// skip verification the same way triggerIngestAPI does for PECHKA_API_URL.
var insecureTransport http.RoundTripper = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

// minioEndpoint strips a scheme from rawURL if present (minio.New expects a
// bare host[:port], not a full URL) and derives whether TLS should be used
// from that scheme. If rawURL has no scheme, it is used as-is and useSSLDefault
// (from MINIO_USE_SSL) decides.
func minioEndpoint(rawURL string, useSSLDefault bool) (string, bool) {
	if u, err := url.Parse(rawURL); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Host, u.Scheme == "https"
	}
	return rawURL, useSSLDefault
}

func uploadMKVToMinIO(ctx context.Context, cfg Config, localDir, discLabel string) ([]MkvFile, error) {
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

		discIndex, err := resolveDiscIndex(cfg.Device)
		if err != nil {
			log.Fatalf("failed to resolve MakeMKV disc index for device %s: %v", cfg.Device, err)
		}

		log.Printf("Extracting disc: %s (MakeMKV disc:%d)", discLabel, discIndex)
		cmd := exec.Command("makemkvcon", "mkv", fmt.Sprintf("disc:%d", discIndex), "all", outputDir)
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

	if discLabel != "" {
		log.Printf("Triggering Ingest API for disc label: %s", discLabel)
		if err := triggerIngestAPI(ctx, cfg.PechkaAPIURL, discLabel); err != nil {
			log.Printf("WARNING: failed to trigger Ingest API: %v", err)
		}
	}
}


var drvLineRe = regexp.MustCompile(`^DRV:(\d+),\d+,\d+,\d+,"[^"]*","[^"]*","([^"]*)"$`)

// resolveDiscIndex maps a device path (e.g. /dev/sr1) to the MakeMKV drive
// index reported by `makemkvcon info`. MakeMKV enumerates its own drive
// slots independently of device naming (e.g. a virtual QEMU DVD-ROM can take
// slot 0 while the real drive ends up in slot 1), so the index used for the
// "disc:N" argument cannot be assumed to match the device's position.
func resolveDiscIndex(device string) (int, error) {
	out, _ := exec.Command("makemkvcon", "-r", "info", "disc:9999").CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		m := drvLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		if m[2] == device {
			return strconv.Atoi(m[1])
		}
	}
	return -1, fmt.Errorf("no MakeMKV drive found for device %q", device)
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
