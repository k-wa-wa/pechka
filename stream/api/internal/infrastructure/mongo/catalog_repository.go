package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"pechka/streaming-service/api/internal/domain"
)

type catalogRepository struct {
	collection *mongo.Collection
}

func NewCatalogRepository(db *mongo.Database) domain.CatalogRepository {
	return &catalogRepository{
		collection: db.Collection("contents"),
	}
}

func (r *catalogRepository) Upsert(ctx context.Context, content *domain.CatalogContent) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": content.ID}
	update := bson.M{"$set": content}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert content to mongo: %w", err)
	}
	return nil
}

func (r *catalogRepository) GetByShortID(ctx context.Context, shortID string, userGroups []string) (*domain.CatalogContent, error) {
	var content domain.CatalogContent
	filter := bson.M{
		"short_id": shortID,
		"$or": []bson.M{
			{"visibility": "public"},
			{"allowed_groups": bson.M{"$in": userGroups}},
		},
	}
	err := r.collection.FindOne(ctx, filter).Decode(&content)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("content not found or access denied")
		}
		return nil, fmt.Errorf("failed to get content from mongo: %w", err)
	}
	return &content, nil
}

func (r *catalogRepository) Search(ctx context.Context, query string, userGroups []string) ([]*domain.CatalogContent, error) {
	// Simple text search or regex for MVP
	titleFilter := bson.M{}
	if query != "" {
		titleFilter = bson.M{"title": bson.M{"$regex": query, "$options": "i"}}
	}
	filter := bson.M{
		"$and": []bson.M{
			titleFilter,
			{
				"$or": []bson.M{
					{"visibility": "public"},
					{"allowed_groups": bson.M{"$in": userGroups}},
				},
			},
		},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search in mongo: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*domain.CatalogContent
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}
	return results, nil
}

func (r *catalogRepository) GetByIDs(ctx context.Context, ids []string, userGroups []string) ([]*domain.CatalogContent, error) {
	if len(ids) == 0 {
		return []*domain.CatalogContent{}, nil
	}

	filter := bson.M{
		"_id": bson.M{"$in": ids},
		"$or": []bson.M{
			{"visibility": "public"},
			{"allowed_groups": bson.M{"$in": userGroups}},
		},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch multiple contents from mongo: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*domain.CatalogContent
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results from mongo: %w", err)
	}

	return results, nil
}
