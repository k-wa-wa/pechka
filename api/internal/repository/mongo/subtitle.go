package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type SubtitleRepository struct {
	col *mongo.Collection
}

func NewSubtitleRepository(db *mongo.Database) *SubtitleRepository {
	return &SubtitleRepository{col: db.Collection("subtitles")}
}

// GetByShortIDAndLanguage は published の字幕トラックのみを返す。
// draft トラックは Benthos の同期対象外のため、そもそもこのコレクションに存在しない。
func (r *SubtitleRepository) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	var s domain.MongoSubtitle
	err := r.col.FindOne(ctx, bson.M{"_id": shortID + ":" + language}).Decode(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
