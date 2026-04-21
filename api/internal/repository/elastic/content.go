package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

const indexName = "stream_contents"

type ContentRepository struct {
	client *elasticsearch.Client
}

func NewContentRepository(client *elasticsearch.Client) *ContentRepository {
	return &ContentRepository{client: client}
}

type SearchResult struct {
	ShortID     string             `json:"short_id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	ContentType domain.ContentType `json:"content_type"`
	Tags        []string           `json:"tags"`
	Status      domain.ContentStatus `json:"status"`
}

func (r *ContentRepository) Search(ctx context.Context, query string, limit, offset int) ([]*SearchResult, error) {
	body := map[string]any{
		"from": offset,
		"size": limit,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"title^3", "description", "tags^2"},
			},
		},
		"_source": []string{"short_id", "title", "description", "content_type", "tags", "status"},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, err
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(indexName),
		r.client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	var esResponse struct {
		Hits struct {
			Hits []struct {
				Source SearchResult `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, err
	}

	results := make([]*SearchResult, 0, len(esResponse.Hits.Hits))
	for i := range esResponse.Hits.Hits {
		results = append(results, &esResponse.Hits.Hits[i].Source)
	}
	return results, nil
}
