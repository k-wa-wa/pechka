package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioConfig struct {
	Bucket    string
	URL       string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func minioConfigFromEnv() MinioConfig {
	return MinioConfig{
		Bucket:    os.Getenv("MINIO_BUCKET"),
		URL:       os.Getenv("MINIO_URL"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	}
}

func downloadMKV(ctx context.Context, mcfg MinioConfig, objectKey, localPath string) error {
	minioClient, err := minio.New(mcfg.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(mcfg.AccessKey, mcfg.SecretKey, ""),
		Secure: mcfg.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	log.Printf("Downloading MKV from MinIO object key: %s...", objectKey)
	err = minioClient.FGetObject(ctx, mcfg.Bucket, objectKey, localPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download object %s: %w", objectKey, err)
	}
	log.Printf("MKV downloaded to local path: %s", localPath)
	return nil
}

func uploadHLSDir(ctx context.Context, mcfg MinioConfig, localDir, shortID string) error {
	minioClient, err := minio.New(mcfg.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(mcfg.AccessKey, mcfg.SecretKey, ""),
		Secure: mcfg.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	files, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("failed to read local output dir: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		localFilePath := filepath.Join(localDir, f.Name())
		objectKey := fmt.Sprintf("resources/hls/%s/%s", shortID, f.Name())

		var contentType string
		if strings.HasSuffix(f.Name(), ".m3u8") {
			contentType = "application/x-mpegURL"
		} else if strings.HasSuffix(f.Name(), ".ts") {
			contentType = "video/MP2T"
		} else {
			contentType = "application/octet-stream"
		}

		log.Printf("Uploading HLS file %s as %s...", f.Name(), objectKey)
		_, err = minioClient.FPutObject(ctx, mcfg.Bucket, objectKey, localFilePath, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("failed to upload HLS file %s: %w", f.Name(), err)
		}
		// 物理ディスクへの瞬間的な書き込み集中（IOPS飽和）を防ぐためのスロットリング
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}


const masterPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS="avc1.640028,mp4a.40.2"
original.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=6192000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8
`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "to-hls" {
		fmt.Fprintf(os.Stderr, "Usage: %s to-hls -input <mkv-object-key> -output <local-dir> -mode video|audio -short-id <id>\n", os.Args[0])
		os.Exit(1)
	}

	fs := flag.NewFlagSet("to-hls", flag.ExitOnError)
	input := fs.String("input", "", "input MKV object key on MinIO")
	output := fs.String("output", "", "local output directory for HLS")
	mode := fs.String("mode", "", "transcode mode: video or audio")
	shortID := fs.String("short-id", "", "content short ID")
	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if *input == "" || *output == "" || *mode == "" || *shortID == "" {
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	mcfg := minioConfigFromEnv()

	// Download MKV from MinIO to local temp file
	tempDir, err := os.MkdirTemp("", "transcode-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	localMKVPath := filepath.Join(tempDir, "input.mkv")
	if err := downloadMKV(ctx, mcfg, *input, localMKVPath); err != nil {
		log.Fatalf("failed to download input MKV: %v", err)
	}

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	// 映像トラックの有無とコーデックを検出
	hasVideo, codec, err := detectVideoMetadata(localMKVPath)
	if err != nil {
		log.Fatalf("failed to detect video metadata: %v", err)
	}
	log.Printf("Video metadata: hasVideo=%t, codec=%s", hasVideo, codec)

	switch *mode {
	case "audio":
		if err := transcodeAudio(localMKVPath, *output); err != nil {
			log.Fatalf("audio transcode failed: %v", err)
		}
	case "original":
		if !hasVideo {
			log.Printf("No video stream found. Skipping original transcode.")
			return
		}
		if err := transcodeOriginal(localMKVPath, *output, codec); err != nil {
			log.Fatalf("original transcode failed: %v", err)
		}
	case "1080p", "720p", "480p":
		if !hasVideo {
			log.Printf("No video stream found. Skipping %s transcode.", *mode)
			return
		}
		if err := transcodeVideoVariant(localMKVPath, *output, *mode); err != nil {
			log.Fatalf("%s transcode failed: %v", *mode, err)
		}
	default:
		log.Fatalf("unknown mode: %q (expected original, 1080p, 720p, 480p or audio)", *mode)
	}

	// Upload HLS directory to MinIO
	log.Printf("Uploading generated HLS variants to MinIO bucket %s prefix resources/%s...", mcfg.Bucket, *shortID)
	if err := uploadHLSDir(ctx, mcfg, *output, *shortID); err != nil {
		log.Fatalf("failed to upload HLS variants to MinIO: %v", err)
	}

	// Clean up HLS local output
	log.Printf("Cleaning up local output HLS directory: %s", *output)
	if err := os.RemoveAll(*output); err != nil {
		log.Printf("WARNING: failed to clean up local output directory: %v", err)
	}

	log.Printf("Transcoding and upload complete for short-id: %s", *shortID)

	// 次のジョブ実行前に物理ディスクの書き込みを落ち着かせるためのクールダウン
	log.Printf("Cooling down disk I/O for 5 seconds...")
	time.Sleep(5 * time.Second)
}

func detectVideoMetadata(input string) (bool, string, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "csv=p=0",
		input,
	)
	out, err := cmd.Output()
	if err != nil {
		// ffprobe がエラー終了した場合は映像なしとする
		return false, "", nil
	}
	codec := strings.TrimSpace(string(out))
	if codec == "" {
		return false, "", nil
	}
	return true, codec, nil
}

func transcodeOriginal(input, outputDir, codec string) error {
	log.Printf("Transcoding original variant (source codec: %s)", codec)
	var args []string
	if codec == "h264" || codec == "hevc" {
		log.Printf("Source video codec %s is supported. Using stream copy.", codec)
		args = []string{
			"-i", input,
			"-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(outputDir, "original_%04d.ts"),
			filepath.Join(outputDir, "original.m3u8"),
		}
	} else {
		log.Printf("Source video codec %s is not directly supported in browser. Re-encoding to H.264.", codec)
		args = []string{
			"-i", input,
			"-c:v", "libx264", "-crf", "18", "-preset", "fast",
			"-c:a", "aac", "-b:a", "192k",
			"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(outputDir, "original_%04d.ts"),
			filepath.Join(outputDir, "original.m3u8"),
		}
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg original: %w", err)
	}
	return nil
}

func transcodeVideoVariant(input, outputDir, resolution string) error {
	log.Printf("Transcoding video variant: %s", resolution)
	var vf, bv, maxrate, bufsize, ba string
	switch resolution {
	case "1080p":
		vf = "scale=1920:1080"
		bv = "6000k"
		maxrate = "6500k"
		bufsize = "12000k"
		ba = "192k"
	case "720p":
		vf = "scale=1280:720"
		bv = "3000k"
		maxrate = "3500k"
		bufsize = "6000k"
		ba = "128k"
	case "480p":
		vf = "scale=854:480"
		bv = "1500k"
		maxrate = "2000k"
		bufsize = "3000k"
		ba = "128k"
	default:
		return fmt.Errorf("unsupported resolution: %s", resolution)
	}

	args := []string{
		"-i", input,
		"-vf", vf, "-c:v", "libx264", "-preset", "fast",
		"-b:v", bv, "-maxrate", maxrate, "-bufsize", bufsize,
		"-c:a", "aac", "-b:a", ba,
		"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(outputDir, fmt.Sprintf("%s_%%04d.ts", resolution)),
		filepath.Join(outputDir, fmt.Sprintf("%s.m3u8", resolution)),
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg %s: %w", resolution, err)
	}
	return nil
}

func transcodeAudio(input, outputDir string) error {
	log.Printf("Transcoding audio variant")
	args := []string{
		"-i", input,
		"-vn", "-c:a", "aac", "-b:a", "192k",
		"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(outputDir, "audio_%04d.ts"),
		filepath.Join(outputDir, "audio.m3u8"),
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg audio: %w", err)
	}
	return nil
}
