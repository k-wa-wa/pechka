package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

const (
	SampleRate = 48000
	Channels   = 1
)

func runCmd(cmdName string, args ...string) (string, error) {
	cmd := exec.Command(cmdName, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s failed (%w):\n%s", cmdName, err, stderr.String())
	}
	return stdout.String(), nil
}

// RunFFmpeg は ffmpeg コマンドを実行する。
func RunFFmpeg(args ...string) error {
	fullArgs := append([]string{"-y", "-loglevel", "error"}, args...)
	_, err := runCmd("ffmpeg", fullArgs...)
	return err
}

// ProbeDurationMs は ffprobe で音声・動画ファイルの実測尺 (ms) を取得する。
func ProbeDurationMs(path string) (int, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		path,
	}
	out, err := runCmd("ffprobe", args...)
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed for %s: %w", path, err)
	}

	var parsed struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	durFloat, err := strconv.ParseFloat(parsed.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s': %w", parsed.Format.Duration, err)
	}

	return int(math.Round(durFloat * 1000.0)), nil
}

// NormalizeAudio は wav を共通フォーマット (48kHz 1ch PCM 16bit) へ揃え、末尾に指定パディングを付与する。
func NormalizeAudio(src, dst string, padMs int) error {
	filters := []string{
		fmt.Sprintf("aresample=%d", SampleRate),
		"aformat=channel_layouts=mono",
	}
	if padMs > 0 {
		filters = append(filters, fmt.Sprintf("apad=pad_dur=%.3f", float64(padMs)/1000.0))
	}

	args := []string{
		"-i", src,
		"-af", strings.Join(filters, ","),
		"-ar", strconv.Itoa(SampleRate),
		"-ac", strconv.Itoa(Channels),
		"-c:a", "pcm_s16le",
		dst,
	}
	return RunFFmpeg(args...)
}

// Silence は指定尺 (ms) の無音 wav ファイルを作成する。
func Silence(dst string, durationMs int) error {
	args := []string{
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=%d:cl=mono", SampleRate),
		"-t", fmt.Sprintf("%.3f", float64(durationMs)/1000.0),
		"-c:a", "pcm_s16le",
		dst,
	}
	return RunFFmpeg(args...)
}

// ConcatAudio は listFile に記述された複数 wav ファイルを連結する。
func ConcatAudio(listFile, dst string) error {
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c:a", "pcm_s16le",
		dst,
	}
	return RunFFmpeg(args...)
}
