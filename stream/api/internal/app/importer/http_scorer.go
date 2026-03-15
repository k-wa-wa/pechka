package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"pechka/streaming-service/api/internal/domain"
)

type HttpThumbnailScorer struct {
	EndpointURL string
}

func NewHttpThumbnailScorer(url string) *HttpThumbnailScorer {
	return &HttpThumbnailScorer{
		EndpointURL: url,
	}
}

type analyzeRequest struct {
	FilePath string    `json:"file_path"`
	Points   []float64 `json:"points"`
}

func (s *HttpThumbnailScorer) Analyze(ctx context.Context, path string, points []float64) (*domain.ScorerResult, error) {
	reqBody, err := json.Marshal(analyzeRequest{
		FilePath: path,
		Points:   points,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.EndpointURL+"/analyze", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res domain.ScorerResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &res, nil
}
