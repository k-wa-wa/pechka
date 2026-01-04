package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	toHlsCmd := flag.NewFlagSet("to-hls", flag.ExitOnError)
	toHlsCmdInput := toHlsCmd.String("input", "", "Path to the input MKV file")
	toHlsCmdOutput := toHlsCmd.String("output", "", "Directory to save the HLS output")
	toHlsCmdMode := toHlsCmd.String("mode", "video", "Transformation mode: video/audio")

	if len(os.Args) < 2 {
		log.Fatal("Subcommand is required")
	}

	switch os.Args[1] {
	case "to-hls":
		toHlsCmd.Parse(os.Args[2:])
		if *toHlsCmdInput == "" || *toHlsCmdOutput == "" {
			log.Fatal("Input MKV file path and output directory are required")
		}
		if err := transformMkv2Hls(*toHlsCmdInput, *toHlsCmdOutput, *toHlsCmdMode); err != nil {
			log.Fatalf("Error transforming MKV to HLS: %v", err)
		}

	default:
		log.Fatalf("Unknown subcommand: %s\n", os.Args[1])
	}
}

func transformMkv2Hls(mkvFilepath string, outputDir string, mode string) error {
	mkvFilename := strings.TrimSuffix(filepath.Base(mkvFilepath), filepath.Ext(mkvFilepath))
	outputPath := filepath.Join(outputDir, mkvFilename+".m3u8")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	args := []string{"-i", mkvFilepath}
	switch mode {
	case "video":
		args = append(
			args,
			"-codec:v", "libx264",
			"-codec:a", "aac",
		)
	case "audio":
		args = append(
			args,
			"-vn",
			"-codec:a", "aac",
			"-b:a", "128k",
		)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
	args = append(
		args,
		"-hls_list_size", "0",
		"-format", "hls",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
