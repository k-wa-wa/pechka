package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultPadMs           = 250
	RequestTimeoutSec      = 120
	DefaultSpeakerID       = 888753760
	DefaultSpeedScale      = 1.25
	DefaultIntonationScale = 1.15
	MockMsPerChar          = 145
	MockBaseMs             = 400
)

type Synthesizer interface {
	Synthesize(text, dst string) error
}

type VoicevoxSynthesizer struct {
	baseURL    string
	speaker    int
	speed      float64
	intonation float64
	client     *http.Client
}

func NewVoicevoxSynthesizer(baseURL string, speaker int, speed, intonation float64) *VoicevoxSynthesizer {
	return &VoicevoxSynthesizer{
		baseURL:    strings.TrimRight(baseURL, "/"),
		speaker:    speaker,
		speed:      speed,
		intonation: intonation,
		client:     &http.Client{Timeout: RequestTimeoutSec * time.Second},
	}
}

func (v *VoicevoxSynthesizer) Synthesize(text, dst string) error {
	ttsText := NormalizeForTTS(text)

	queryURL := fmt.Sprintf("%s/audio_query?text=%s&speaker=%d", v.baseURL, url.QueryEscape(ttsText), v.speaker)
	req, err := http.NewRequest(http.MethodPost, queryURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create audio_query request: %w", err)
	}

	res, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("audio_query request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("audio_query status %d: %s", res.StatusCode, string(body))
	}

	var params map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&params); err != nil {
		return fmt.Errorf("failed to decode audio_query response: %w", err)
	}

	params["speedScale"] = v.speed
	params["intonationScale"] = v.intonation
	params["prePhonemeLength"] = 0.0
	params["postPhonemeLength"] = 0.0

	synthURL := fmt.Sprintf("%s/synthesis?speaker=%d", v.baseURL, v.speaker)
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	reqSynth, err := http.NewRequest(http.MethodPost, synthURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create synthesis request: %w", err)
	}
	reqSynth.Header.Set("Content-Type", "application/json")

	resSynth, err := v.client.Do(reqSynth)
	if err != nil {
		return fmt.Errorf("synthesis request failed: %w", err)
	}
	defer resSynth.Body.Close()

	if resSynth.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resSynth.Body)
		return fmt.Errorf("synthesis status %d: %s", resSynth.StatusCode, string(body))
	}

	audioData, err := io.ReadAll(resSynth.Body)
	if err != nil {
		return fmt.Errorf("failed to read audio data: %w", err)
	}

	return os.WriteFile(dst, audioData, 0o644)
}

type MockSynthesizer struct{}

func (m *MockSynthesizer) Synthesize(text, dst string) error {
	durMs := MockBaseMs + len([]rune(text))*MockMsPerChar
	return Silence(dst, durMs)
}

func BuildSynthesizer(engine, engineURL string, speaker int, speed, intonation float64) (Synthesizer, error) {
	switch engine {
	case "mock":
		return &MockSynthesizer{}, nil
	case "voicevox":
		return NewVoicevoxSynthesizer(engineURL, speaker, speed, intonation), nil
	default:
		return nil, fmt.Errorf("unknown engine: %s (expected 'voicevox' or 'mock')", engine)
	}
}

func RunSynthesize(entries []*Entry, outDir string, synth Synthesizer, padMs int) ([]int, error) {
	audioDir := filepath.Join(outDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create audio dir: %w", err)
	}

	var durations []int
	for _, entry := range entries {
		rawPath := filepath.Join(audioDir, fmt.Sprintf("%04d.raw.wav", entry.Seq))
		finalPath := filepath.Join(audioDir, fmt.Sprintf("%04d.wav", entry.Seq))

		if err := synth.Synthesize(entry.Text, rawPath); err != nil {
			return nil, fmt.Errorf("synthesize failed for entry %d: %w", entry.Seq, err)
		}

		if err := NormalizeAudio(rawPath, finalPath, padMs); err != nil {
			os.Remove(rawPath)
			return nil, fmt.Errorf("normalize audio failed for entry %d: %w", entry.Seq, err)
		}
		os.Remove(rawPath)

		duration, err := ProbeDurationMs(finalPath)
		if err != nil {
			return nil, fmt.Errorf("probe duration failed for %s: %w", finalPath, err)
		}

		entry.AudioPath = finalPath
		durations = append(durations, duration)
		fmt.Printf("  [%04d] %6.2fs  %s\n", entry.Seq, float64(duration)/1000.0, truncateString(entry.Text, 40))
	}

	return durations, nil
}

func BuildNarrationConcat(entries []*Entry, outDir string) (string, error) {
	listPath := filepath.Join(outDir, "audio", "list.txt")
	var sb strings.Builder
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf("file '%s'\n", entry.AudioPath))
	}
	if err := os.WriteFile(listPath, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}

	dstPath := filepath.Join(outDir, "narration.wav")
	if err := ConcatAudio(listPath, dstPath); err != nil {
		return "", err
	}
	return dstPath, nil
}

func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
