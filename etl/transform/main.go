package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

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
		fmt.Fprintf(os.Stderr, "Usage: %s to-hls -input <mkv> -output <dir> -mode video|audio\n", os.Args[0])
		os.Exit(1)
	}

	fs := flag.NewFlagSet("to-hls", flag.ExitOnError)
	input := fs.String("input", "", "input MKV file path")
	output := fs.String("output", "", "output directory (NFS HLS dir)")
	mode := fs.String("mode", "", "transcode mode: video or audio")
	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if *input == "" || *output == "" || *mode == "" {
		fs.Usage()
		os.Exit(1)
	}

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	switch *mode {
	case "video":
		if err := transcodeVideo(*input, *output); err != nil {
			log.Fatalf("video transcode failed: %v", err)
		}
	case "audio":
		if err := transcodeAudio(*input, *output); err != nil {
			log.Fatalf("audio transcode failed: %v", err)
		}
	default:
		log.Fatalf("unknown mode: %q (expected video or audio)", *mode)
	}
}

type variantSpec struct {
	name string
	args []string
}

func transcodeVideo(input, outputDir string) error {
	variants := []variantSpec{
		{
			name: "original",
			args: []string{
				"-i", input,
				"-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
				"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
				"-hls_segment_filename", filepath.Join(outputDir, "original_%04d.ts"),
				filepath.Join(outputDir, "original.m3u8"),
			},
		},
		{
			name: "1080p",
			args: []string{
				"-i", input,
				"-vf", "scale=1920:1080", "-c:v", "libx264", "-preset", "fast",
				"-b:v", "6000k", "-maxrate", "6500k", "-bufsize", "12000k",
				"-c:a", "aac", "-b:a", "192k",
				"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
				"-hls_segment_filename", filepath.Join(outputDir, "1080p_%04d.ts"),
				filepath.Join(outputDir, "1080p.m3u8"),
			},
		},
		{
			name: "720p",
			args: []string{
				"-i", input,
				"-vf", "scale=1280:720", "-c:v", "libx264", "-preset", "fast",
				"-b:v", "3000k", "-maxrate", "3500k", "-bufsize", "6000k",
				"-c:a", "aac", "-b:a", "128k",
				"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
				"-hls_segment_filename", filepath.Join(outputDir, "720p_%04d.ts"),
				filepath.Join(outputDir, "720p.m3u8"),
			},
		},
		{
			name: "480p",
			args: []string{
				"-i", input,
				"-vf", "scale=854:480", "-c:v", "libx264", "-preset", "fast",
				"-b:v", "1500k", "-maxrate", "2000k", "-bufsize", "3000k",
				"-c:a", "aac", "-b:a", "128k",
				"-f", "hls", "-hls_time", "6", "-hls_list_size", "0",
				"-hls_segment_filename", filepath.Join(outputDir, "480p_%04d.ts"),
				filepath.Join(outputDir, "480p.m3u8"),
			},
		},
	}

	for _, v := range variants {
		log.Printf("Transcoding variant: %s", v.name)
		cmd := exec.Command("ffmpeg", v.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg %s: %w", v.name, err)
		}
	}

	masterPath := filepath.Join(outputDir, "master.m3u8")
	if err := os.WriteFile(masterPath, []byte(masterPlaylistTemplate), 0644); err != nil {
		return fmt.Errorf("write master playlist: %w", err)
	}
	log.Printf("Master playlist written to %s", masterPath)
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
