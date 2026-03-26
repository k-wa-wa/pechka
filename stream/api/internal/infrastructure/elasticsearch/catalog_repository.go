package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"pechka/streaming-service/api/internal/domain"
)

type searchRepository struct {
	client *elasticsearch.Client
}

func NewSearchRepository(client *elasticsearch.Client) domain.SearchRepository {
	return &searchRepository{
		client: client,
	}
}

func (r *searchRepository) SearchIDs(ctx context.Context, query string, tags []string, userGroups []string) ([]string, error) {
	var buf bytes.Buffer
	
	// Complex query construction
	queryMap := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
				"filter": []interface{}{},
			},
		},
		"_source": false, // Only need IDs
		"size": 100,      // Limit results
	}

	boolQuery := queryMap["query"].(map[string]interface{})["bool"].(map[string]interface{})

	if query != "" {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"title^3", "description"},
			},
		})
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
				"term": map[string]interface{}{
					"tags": tag,
				},
			})
		}
	}

	// RBAC Filter: visibility == "public" OR allowed_groups IN userGroups
	rbacFilter := map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{"term": map[string]interface{}{"visibility.keyword": "public"}},
			},
			"minimum_should_match": 1,
		},
	}

	if len(userGroups) > 0 {
		rbacFilter["bool"].(map[string]interface{})["should"] = append(
			rbacFilter["bool"].(map[string]interface{})["should"].([]interface{}),
			map[string]interface{}{
				"terms": map[string]interface{}{"allowed_groups.keyword": userGroups},
			},
		)
	}

	boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), rbacFilter)


	if err := json.NewEncoder(&buf).Encode(queryMap); err != nil {
		return nil, fmt.Errorf("failed to encode ES query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex("contents"),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ES search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("error searching: %s", res.Status())
		}
		return nil, fmt.Errorf("error searching [%s]: %v", res.Status(), e)
	}

	var rMap map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&rMap); err != nil {
		return nil, fmt.Errorf("failed to decode ES response: %w", err)
	}

	hits := rMap["hits"].(map[string]interface{})["hits"].([]interface{})
	ids := make([]string, 0, len(hits))
	for _, hit := range hits {
		source := hit.(map[string]interface{})
		ids = append(ids, source["_id"].(string))
	}

	return ids, nil
}
