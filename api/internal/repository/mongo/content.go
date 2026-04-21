package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type ContentRepository struct {
	col *mongo.Collection
}

func NewContentRepository(db *mongo.Database) *ContentRepository {
	return &ContentRepository{col: db.Collection("contents")}
}

type ListParams struct {
	ContentType *domain.ContentType
	Status      *domain.ContentStatus
	Limit       int64
	Offset      int64
}

func (r *ContentRepository) List(ctx context.Context, params ListParams) ([]*domain.MongoContent, error) {
	filter := bson.M{}
	if params.ContentType != nil {
		filter["content_type"] = *params.ContentType
	}
	if params.Status != nil {
		filter["status"] = *params.Status
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(params.Limit).
		SetSkip(params.Offset)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var contents []*domain.MongoContent
	if err := cursor.All(ctx, &contents); err != nil {
		return nil, err
	}
	return contents, nil
}

func (r *ContentRepository) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	var c domain.MongoContent
	err := r.col.FindOne(ctx, bson.M{"_id": shortID}).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContentRepository) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	var c domain.MongoContent
	opts := options.FindOne().SetProjection(bson.M{"variants": 1})
	err := r.col.FindOne(ctx, bson.M{"_id": shortID}, opts).Decode(&c)
	if err != nil {
		return nil, err
	}
	return c.Variants, nil
}
