package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const SegmentSeconds = 6

type Variant struct {
	Name       string
	Bandwidth  int
	Resolution string
	Codecs     string
}

var Variants = []Variant{
	{Name: "original", Bandwidth: 0, Resolution: "", Codecs: ""},
	{Name: "720p", Bandwidth: 3128000, Resolution: "1280x720", Codecs: "avc1.4d001f,mp4a.40.2"},
}

func TranscodeHLS(mp4Path, outDir string) ([]Variant, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	for _, v := range Variants {
		var args []string
		if v.Name == "original" {
			args = []string{"-i", mp4Path, "-c:v", "copy", "-c:a", "aac", "-b:a", "192k"}
		} else {
			args = []string{
				"-i", mp4Path,
				"-vf", "scale=1280:720",
				"-c:v", "libx264", "-preset", "fast",
				"-b:v", "3000k", "-maxrate", "3500k", "-bufsize", "6000k",
				"-c:a", "aac", "-b:a", "128k",
			}
		}

		hlsArgs := []string{
			"-f", "hls",
			"-hls_time", fmt.Sprintf("%d", SegmentSeconds),
			"-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(outDir, fmt.Sprintf("%s_%%04d.ts", v.Name)),
			filepath.Join(outDir, fmt.Sprintf("%s.m3u8", v.Name)),
		}

		fullArgs := append(args, hlsArgs...)
		if err := RunFFmpeg(fullArgs...); err != nil {
			return nil, fmt.Errorf("transcode failed for variant %s: %w", v.Name, err)
		}
		fmt.Printf("  transcoded %s\n", v.Name)
	}

	return Variants, nil
}

func MasterPlaylist(variants []Variant) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n\n")

	for _, v := range variants {
		if v.Name == "original" {
			sb.WriteString("#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS=\"avc1.640028,mp4a.40.2\"\n")
		} else {
			sb.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,CODECS=\"%s\"\n",
				v.Bandwidth, v.Resolution, v.Codecs))
		}
		sb.WriteString(fmt.Sprintf("%s.m3u8\n\n", v.Name))
	}

	return sb.String()
}

func GenerateThumbnail(mp4Path, dstPath string, atSeconds float64) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return "", err
	}

	args := []string{
		"-ss", fmt.Sprintf("%.2f", atSeconds),
		"-i", mp4Path,
		"-frames:v", "1",
		"-q:v", "3",
		dstPath,
	}
	if err := RunFFmpeg(args...); err != nil {
		return "", fmt.Errorf("generate thumbnail failed: %w", err)
	}
	return dstPath, nil
}
