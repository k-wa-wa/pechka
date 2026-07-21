package cmd

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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/k-wa-wa/pechka/batch/etl/shared"
)

func downloadMKV(ctx context.Context, mcfg shared.MinioConfig, objectKey, localPath string) error {
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

func uploadHLSDir(ctx context.Context, mcfg shared.MinioConfig, localDir, shortID string) error {
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
		time.Sleep(50 * time.Millisecond)
	}
	return nil
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

func RunTransform(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("transform", flag.ContinueOnError)
	input := fs.String("input", "", "input MKV object key on MinIO")
	output := fs.String("output", "", "local output directory for HLS")
	mode := fs.String("mode", "", "transcode mode (original, 1080p, 720p, 480p, audio)")
	shortID := fs.String("short-id", "", "content short ID")
	contentID := fs.String("content-id", "", "content ID in database")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	if *input == "" || *output == "" || *mode == "" || *shortID == "" {
		fs.Usage()
		return fmt.Errorf("missing required arguments for transform")
	}

	mcfg := shared.MinioConfigFromEnv()

	tempDir, err := os.MkdirTemp("", "transcode-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	localMKVPath := filepath.Join(tempDir, "input.mkv")
	if err := downloadMKV(ctx, mcfg, *input, localMKVPath); err != nil {
		return fmt.Errorf("failed to download input MKV: %w", err)
	}

	if err := os.MkdirAll(*output, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	hasVideo, codec, err := detectVideoMetadata(localMKVPath)
	if err != nil {
		return fmt.Errorf("failed to detect video metadata: %w", err)
	}
	log.Printf("Video metadata: hasVideo=%t, codec=%s", hasVideo, codec)

	switch *mode {
	case "audio":
		if err := transcodeAudio(localMKVPath, *output); err != nil {
			return fmt.Errorf("audio transcode failed: %w", err)
		}
	case "original":
		if !hasVideo {
			log.Printf("No video stream found. Skipping original transcode.")
			return nil
		}
		if err := transcodeOriginal(localMKVPath, *output, codec); err != nil {
			return fmt.Errorf("original transcode failed: %w", err)
		}
	case "1080p", "720p", "480p":
		if !hasVideo {
			log.Printf("No video stream found. Skipping %s transcode.", *mode)
			return nil
		}
		if err := transcodeVideoVariant(localMKVPath, *output, *mode); err != nil {
			return fmt.Errorf("%s transcode failed: %w", *mode, err)
		}
	default:
		return fmt.Errorf("unknown mode: %q (expected original, 1080p, 720p, 480p or audio)", *mode)
	}

	log.Printf("Uploading generated HLS variants to MinIO bucket %s prefix resources/%s...", mcfg.Bucket, *shortID)
	if err := uploadHLSDir(ctx, mcfg, *output, *shortID); err != nil {
		return fmt.Errorf("failed to upload HLS variants to MinIO: %w", err)
	}

	log.Printf("Transcoding and upload complete for short-id: %s", *shortID)

	log.Printf("Cleaning up local output HLS directory: %s", *output)
	if err := os.RemoveAll(*output); err != nil {
		log.Printf("WARNING: failed to clean up local output directory: %v", err)
	}

	postgresDSN := shared.GetPostgresDSN()
	if postgresDSN == "" {
		log.Printf("PostgreSQL connection string is not set (DB_HOST empty). Skipping database registration.")
	} else if *contentID == "" {
		log.Printf("Content ID is not provided. Skipping database registration.")
	} else {
		log.Printf("Connecting to PostgreSQL to register variant...")
		db, err := pgxpool.New(ctx, postgresDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		defer db.Close()

		vSpec := shared.BuildVariantInfo(*mode)
		
		log.Printf("Registering variant %s to content %s...", *mode, *contentID)
		if err := shared.RegisterVariant(ctx, db, *contentID, *shortID, *mode, vSpec); err != nil {
			return fmt.Errorf("failed to register variant %s: %w", *mode, err)
		}

		minioClient, err := minio.New(mcfg.URL, &minio.Options{
			Creds:  credentials.NewStaticV4(mcfg.AccessKey, mcfg.SecretKey, ""),
			Secure: mcfg.UseSSL,
		})
		if err != nil {
			return fmt.Errorf("failed to create MinIO client for master playlist: %w", err)
		}

		variants, err := shared.GetMinioVariants(ctx, minioClient, mcfg.Bucket, *shortID)
		if err != nil {
			return fmt.Errorf("failed to get MinIO HLS variants: %w", err)
		}

		createdMaster, err := shared.GenerateAndUploadMasterPlaylist(ctx, minioClient, mcfg.Bucket, *shortID, variants)
		if err != nil {
			return fmt.Errorf("failed to generate and upload master playlist: %w", err)
		}

		if createdMaster {
			log.Printf("Registering/updating master variant in database...")
			masterSpec := shared.VariantInfo{VariantType: "master"}
			if err := shared.RegisterVariant(ctx, db, *contentID, *shortID, "master", masterSpec); err != nil {
				return fmt.Errorf("failed to register master variant: %w", err)
			}
		}
	}

	log.Printf("Cooling down disk I/O for 5 seconds...")
	time.Sleep(5 * time.Second)
	return nil
}
