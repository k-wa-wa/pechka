package elastic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestContentRepository_Search(t *testing.T) {
	t.Run("successful search", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"hits": {
					"hits": [
						{
							"_source": {
								"short_id": "c1",
								"title": "Title 1",
								"description": "Desc 1",
								"content_type": "video",
								"channel_id": "ch1",
								"channel_name": "Channel 1",
								"tags": ["go", "test"],
								"status": "ready"
							}
						}
					]
				}
			}`))
		}))
		defer ts.Close()

		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{ts.URL},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		repo := NewContentRepository(client)
		results, err := repo.Search(context.Background(), "test", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].ShortID != "c1" || results[0].Title != "Title 1" || results[0].ContentType != domain.ContentTypeVideo {
			t.Errorf("unexpected content source: %+v", results[0])
		}
	})

	t.Run("elasticsearch error response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal error"}`))
		}))
		defer ts.Close()

		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{ts.URL},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		repo := NewContentRepository(client)
		_, err = repo.Search(context.Background(), "test", 10, 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
