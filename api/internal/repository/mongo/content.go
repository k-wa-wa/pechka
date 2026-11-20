package mongo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

var (
	ErrUnmigratedReadModel = errors.New("mongodb contents read model is not migrated")
	ErrInvalidCursor       = errors.New("invalid cursor")
)

type ContentRepository struct {
	col            *mongo.Collection
	indexesEnsured atomic.Bool
	migrated       atomic.Bool
}

func NewContentRepository(db *mongo.Database) *ContentRepository {
	return &ContentRepository{col: db.Collection("contents")}
}

type ListParams struct {
	ContentType *domain.ContentType
	ChannelID   *string
	Status      *domain.ContentStatus
	Limit       int64
	Offset      int64
}

// archived_at と deleted_at が未設定/nullのドキュメントのみを対象とすることで、
// admin画面以外(公開API)からアーカイブ済み・論理削除済みコンテンツを見えなくする。
func (r *ContentRepository) List(ctx context.Context, params ListParams) ([]*domain.MongoContent, error) {
	filter := bson.M{"archived_at": nil, "deleted_at": nil}
	if params.ContentType != nil {
		filter["content_type"] = *params.ContentType
	}
	if params.ChannelID != nil {
		filter["channel_id"] = *params.ChannelID
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
	err := r.col.FindOne(ctx, bson.M{"_id": shortID, "archived_at": nil, "deleted_at": nil}).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContentRepository) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	var c domain.MongoContent
	opts := options.FindOne().SetProjection(bson.M{"variants": 1})
	err := r.col.FindOne(ctx, bson.M{"_id": shortID, "archived_at": nil, "deleted_at": nil}, opts).Decode(&c)
	if err != nil {
		return nil, err
	}
	return c.Variants, nil
}

type AdminListParams struct {
	Status    *domain.ContentStatus
	ChannelID *string
	Sort      domain.ContentSort
	After     *AdminCursor
	Before    *AdminCursor
	Limit     int
}

type AdminCursor struct {
	Sort  domain.ContentSort
	Time  time.Time
	Title string
	ID    string
}

type rawAdminCursor struct {
	Sort domain.ContentSort `json:"sort"`
	V    json.RawMessage    `json:"v"`
	ID   string             `json:"id"`
}

func isTimeSort(sort domain.ContentSort) bool {
	switch sort {
	case domain.ContentSortUpdatedDesc,
		domain.ContentSortUpdatedAsc,
		domain.ContentSortCreatedDesc,
		domain.ContentSortCreatedAsc:
		return true
	default:
		return false
	}
}

func sortFieldAndDir(sort domain.ContentSort) (string, int, error) {
	switch sort {
	case domain.ContentSortUpdatedDesc:
		return "updated_at", -1, nil
	case domain.ContentSortUpdatedAsc:
		return "updated_at", 1, nil
	case domain.ContentSortCreatedDesc:
		return "created_at", -1, nil
	case domain.ContentSortCreatedAsc:
		return "created_at", 1, nil
	case domain.ContentSortTitleAsc:
		return "title", 1, nil
	case domain.ContentSortTitleDesc:
		return "title", -1, nil
	default:
		return "", 0, fmt.Errorf("invalid sort: %q", sort)
	}
}

func EncodeAdminCursor(sort domain.ContentSort, c *domain.Content) (string, error) {
	if c == nil || c.ShortID == "" {
		return "", fmt.Errorf("%w: missing content or short_id", ErrInvalidCursor)
	}
	if !domain.IsValidContentSort(sort) {
		return "", fmt.Errorf("%w: invalid sort %q", ErrInvalidCursor, sort)
	}

	var value any
	switch sort {
	case domain.ContentSortUpdatedDesc, domain.ContentSortUpdatedAsc:
		value = c.UpdatedAt.UnixMilli()
	case domain.ContentSortCreatedDesc, domain.ContentSortCreatedAsc:
		value = c.CreatedAt.UnixMilli()
	case domain.ContentSortTitleAsc, domain.ContentSortTitleDesc:
		value = c.Title
	}

	payload, err := json.Marshal(struct {
		Sort domain.ContentSort `json:"sort"`
		V    any                `json:"v"`
		ID   string             `json:"id"`
	}{
		Sort: sort,
		V:    value,
		ID:   c.ShortID,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeAdminCursor(token string, expectedSort domain.ContentSort) (*AdminCursor, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: empty token", ErrInvalidCursor)
	}
	rawBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("%w: base64 decode failed", ErrInvalidCursor)
	}

	var raw rawAdminCursor
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal failed", ErrInvalidCursor)
	}
	if !domain.IsValidContentSort(raw.Sort) {
		return nil, fmt.Errorf("%w: invalid sort in cursor", ErrInvalidCursor)
	}
	if expectedSort != "" && raw.Sort != expectedSort {
		return nil, fmt.Errorf("%w: sort mismatch", ErrInvalidCursor)
	}
	if raw.ID == "" {
		return nil, fmt.Errorf("%w: missing id", ErrInvalidCursor)
	}
	if len(raw.V) == 0 || string(raw.V) == "null" {
		return nil, fmt.Errorf("%w: missing v", ErrInvalidCursor)
	}

	cur := &AdminCursor{
		Sort: raw.Sort,
		ID:   raw.ID,
	}
	if isTimeSort(raw.Sort) {
		var ms int64
		if err := json.Unmarshal(raw.V, &ms); err != nil {
			return nil, fmt.Errorf("%w: invalid timestamp value", ErrInvalidCursor)
		}
		cur.Time = time.UnixMilli(ms).UTC()
	} else {
		var title string
		if err := json.Unmarshal(raw.V, &title); err != nil {
			return nil, fmt.Errorf("%w: invalid title value", ErrInvalidCursor)
		}
		cur.Title = title
	}
	return cur, nil
}

// AdminIndexModels は管理画面一覧の 6 種類のソートに対応する複合インデックス定義を返す。
func AdminIndexModels() []mongo.IndexModel {
	specs := []struct {
		name  string
		field string
		dir   int32
	}{
		{name: "idx_admin_updated_desc", field: "updated_at", dir: -1},
		{name: "idx_admin_updated_asc", field: "updated_at", dir: 1},
		{name: "idx_admin_created_desc", field: "created_at", dir: -1},
		{name: "idx_admin_created_asc", field: "created_at", dir: 1},
		{name: "idx_admin_title_asc", field: "title", dir: 1},
		{name: "idx_admin_title_desc", field: "title", dir: -1},
	}
	models := make([]mongo.IndexModel, 0, len(specs))
	for _, s := range specs {
		models = append(models, mongo.IndexModel{
			Keys: bson.D{
				{Key: s.field, Value: s.dir},
				{Key: "_id", Value: int32(1)},
			},
			Options: options.Index().SetName(s.name),
		})
	}
	return models
}

// EnsureIndexes は管理一覧用複合インデックスを冪等に作成する。
func (r *ContentRepository) EnsureIndexes(ctx context.Context) error {
	if r.col == nil {
		return errors.New("mongo collection is nil")
	}
	if _, err := r.col.Indexes().CreateMany(ctx, AdminIndexModels()); err != nil {
		return err
	}
	r.indexesEnsured.Store(true)
	return nil
}

func unmigratedAdminDocFilter() bson.M {
	return bson.M{
		"$or": bson.A{
			bson.M{"id": bson.M{"$exists": false}},
			bson.M{"id": nil},
			bson.M{"id": ""},
			bson.M{"id": bson.M{"$not": bson.M{"$type": "string"}}},
			bson.M{"created_at": bson.M{"$not": bson.M{"$type": "date"}}},
			bson.M{"updated_at": bson.M{"$not": bson.M{"$type": "date"}}},
			bson.M{"$and": bson.A{
				bson.M{"published_at": bson.M{"$exists": true, "$ne": nil}},
				bson.M{"published_at": bson.M{"$not": bson.M{"$type": "date"}}},
			}},
			bson.M{"$and": bson.A{
				bson.M{"archived_at": bson.M{"$exists": true, "$ne": nil}},
				bson.M{"archived_at": bson.M{"$not": bson.M{"$type": "date"}}},
			}},
			bson.M{"deleted_at": bson.M{"$exists": false}},
			bson.M{"$and": bson.A{
				bson.M{"deleted_at": bson.M{"$ne": nil}},
				bson.M{"deleted_at": bson.M{"$not": bson.M{"$type": "date"}}},
			}},
		},
	}
}

type adminListQueryPlan struct {
	BaseFilter bson.M
	FindFilter bson.M
	Sort       bson.D
	FetchLimit int64
	Reverse    bool
	Field      string
	Dir        int
}

func buildAdminRangeFilter(field string, dir int, cur *AdminCursor, isAfter bool) bson.M {
	var k any
	if field == "title" {
		k = cur.Title
	} else {
		k = cur.Time.UTC()
	}

	var cmpOp string
	var idOp string
	if isAfter {
		if dir == 1 {
			cmpOp = "$gt"
		} else {
			cmpOp = "$lt"
		}
		idOp = "$gt"
	} else {
		if dir == 1 {
			cmpOp = "$lt"
		} else {
			cmpOp = "$gt"
		}
		idOp = "$lt"
	}

	return bson.M{
		"$or": bson.A{
			bson.M{field: bson.M{cmpOp: k}},
			bson.M{field: k, "_id": bson.M{idOp: cur.ID}},
		},
	}
}

func combineFilters(baseFilter, rangeFilter bson.M) bson.M {
	if len(rangeFilter) == 0 {
		return baseFilter
	}
	return bson.M{
		"$and": bson.A{
			baseFilter,
			rangeFilter,
		},
	}
}

func buildAdminListQuery(params AdminListParams) (adminListQueryPlan, error) {
	if params.After != nil && params.Before != nil {
		return adminListQueryPlan{}, fmt.Errorf("%w: after and before cannot be specified together", ErrInvalidCursor)
	}
	sort := params.Sort
	if sort == "" {
		sort = domain.ContentSortUpdatedDesc
	}
	field, dir, err := sortFieldAndDir(sort)
	if err != nil {
		return adminListQueryPlan{}, err
	}

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 論理削除済みのみ除外し、アーカイブ済み（archived_at != nil）は管理一覧に含める。
	baseFilter := bson.M{"deleted_at": nil}
	if params.Status != nil {
		baseFilter["status"] = *params.Status
	}
	if params.ChannelID != nil {
		baseFilter["channel_id"] = *params.ChannelID
	}

	plan := adminListQueryPlan{
		BaseFilter: baseFilter,
		FindFilter: baseFilter,
		FetchLimit: int64(limit + 1),
		Field:      field,
		Dir:        dir,
	}

	if params.Before != nil {
		if params.Before.Sort != sort {
			return adminListQueryPlan{}, fmt.Errorf("%w: cursor sort mismatch", ErrInvalidCursor)
		}
		plan.Reverse = true
		plan.FindFilter = combineFilters(baseFilter, buildAdminRangeFilter(field, dir, params.Before, false))
		plan.Sort = bson.D{
			{Key: field, Value: -dir},
			{Key: "_id", Value: -1},
		}
		return plan, nil
	}

	if params.After != nil {
		if params.After.Sort != sort {
			return adminListQueryPlan{}, fmt.Errorf("%w: cursor sort mismatch", ErrInvalidCursor)
		}
		plan.FindFilter = combineFilters(baseFilter, buildAdminRangeFilter(field, dir, params.After, true))
	}

	plan.Sort = bson.D{
		{Key: field, Value: dir},
		{Key: "_id", Value: 1},
	}
	return plan, nil
}

type mongoAdminDoc struct {
	ShortID          string               `bson:"_id"`
	UUID             string               `bson:"id"`
	ContentType      domain.ContentType   `bson:"content_type"`
	ChannelID        string               `bson:"channel_id"`
	DiscID           *string              `bson:"disc_id,omitempty"`
	Title            string               `bson:"title"`
	Description      string               `bson:"description"`
	DurationSeconds  *int                 `bson:"duration_seconds,omitempty"`
	Is360            bool                 `bson:"is_360"`
	Tags             []string             `bson:"tags"`
	Status           domain.ContentStatus `bson:"status"`
	DocumentKey      *string              `bson:"document_key,omitempty"`
	DocumentMimeType *string              `bson:"document_mime_type,omitempty"`
	PublishedAt      *time.Time           `bson:"published_at,omitempty"`
	ArchivedAt       *time.Time           `bson:"archived_at,omitempty"`
	CreatedAt        time.Time            `bson:"created_at"`
	UpdatedAt        time.Time            `bson:"updated_at"`
}

func (d *mongoAdminDoc) toDomain() *domain.Content {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	var publishedAt *time.Time
	if d.PublishedAt != nil {
		t := d.PublishedAt.UTC()
		publishedAt = &t
	}
	var archivedAt *time.Time
	if d.ArchivedAt != nil {
		t := d.ArchivedAt.UTC()
		archivedAt = &t
	}
	return &domain.Content{
		ID:               d.UUID,
		ShortID:          d.ShortID,
		ContentType:      d.ContentType,
		ChannelID:        d.ChannelID,
		DiscID:           d.DiscID,
		Title:            d.Title,
		Description:      d.Description,
		DurationSeconds:  d.DurationSeconds,
		Is360:            d.Is360,
		Tags:             tags,
		Status:           d.Status,
		DocumentKey:      d.DocumentKey,
		DocumentMimeType: d.DocumentMimeType,
		PublishedAt:      publishedAt,
		ArchivedAt:       archivedAt,
		CreatedAt:        d.CreatedAt.UTC(),
		UpdatedAt:        d.UpdatedAt.UTC(),
	}
}

func cursorFromContent(sort domain.ContentSort, c *domain.Content) *AdminCursor {
	return &AdminCursor{
		Sort:  sort,
		Time:  timeForSort(sort, c),
		Title: c.Title,
		ID:    c.ShortID,
	}
}

func timeForSort(sort domain.ContentSort, c *domain.Content) time.Time {
	switch sort {
	case domain.ContentSortCreatedDesc, domain.ContentSortCreatedAsc:
		return c.CreatedAt.UTC()
	default:
		return c.UpdatedAt.UTC()
	}
}

type adminDocStore interface {
	FindUnmigrated(ctx context.Context, filter bson.M) (bool, error)
	CountDocuments(ctx context.Context, filter bson.M) (int64, error)
	FindAdminDocs(ctx context.Context, filter bson.M, sort bson.D, limit int64) ([]mongoAdminDoc, error)
	ExistsAdminDoc(ctx context.Context, filter bson.M) (bool, error)
}

type mongoCollectionAdminStore struct {
	col *mongo.Collection
}

func (s *mongoCollectionAdminStore) FindUnmigrated(ctx context.Context, filter bson.M) (bool, error) {
	var probe bson.M
	err := s.col.FindOne(ctx, filter, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&probe)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (s *mongoCollectionAdminStore) CountDocuments(ctx context.Context, filter bson.M) (int64, error) {
	return s.col.CountDocuments(ctx, filter)
}

func (s *mongoCollectionAdminStore) FindAdminDocs(ctx context.Context, filter bson.M, sort bson.D, limit int64) ([]mongoAdminDoc, error) {
	opts := options.Find().SetSort(sort).SetLimit(limit)
	cursor, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []mongoAdminDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *mongoCollectionAdminStore) ExistsAdminDoc(ctx context.Context, filter bson.M) (bool, error) {
	var probe bson.M
	err := s.col.FindOne(ctx, filter, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&probe)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (r *ContentRepository) ListAdmin(ctx context.Context, params AdminListParams) (*domain.AdminContentPage, error) {
	if r.col == nil {
		return nil, errors.New("mongo collection is nil")
	}
	if !r.indexesEnsured.Load() {
		if err := r.EnsureIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return r.listAdminWithStore(ctx, &mongoCollectionAdminStore{col: r.col}, params)
}

func (r *ContentRepository) listAdminWithStore(ctx context.Context, store adminDocStore, params AdminListParams) (*domain.AdminContentPage, error) {
	sort := params.Sort
	if sort == "" {
		sort = domain.ContentSortUpdatedDesc
	}
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	params.Sort = sort
	params.Limit = limit

	plan, err := buildAdminListQuery(params)
	if err != nil {
		return nil, err
	}

	if !r.migrated.Load() {
		foundUnmigrated, err := store.FindUnmigrated(ctx, unmigratedAdminDocFilter())
		if err != nil {
			return nil, err
		}
		if foundUnmigrated {
			slog.WarnContext(ctx, "unmigrated document detected in mongodb contents collection; falling back to postgres")
			return nil, ErrUnmigratedReadModel
		}
		r.migrated.Store(true)
	}

	total64, err := store.CountDocuments(ctx, plan.BaseFilter)
	if err != nil {
		return nil, err
	}
	total := int(total64)

	docs, err := store.FindAdminDocs(ctx, plan.FindFilter, plan.Sort, plan.FetchLimit)
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return &domain.AdminContentPage{
			Contents: []*domain.Content{},
			Total:    total,
			HasNext:  false,
			HasPrev:  false,
		}, nil
	}

	hasMore := len(docs) > limit
	if hasMore {
		docs = docs[:limit]
	}

	contents := make([]*domain.Content, 0, len(docs))
	for i := range docs {
		contents = append(contents, docs[i].toDomain())
	}

	if plan.Reverse {
		slices.Reverse(contents)
	}

	var hasNext, hasPrev bool
	if plan.Reverse {
		hasPrev = hasMore
		last := contents[len(contents)-1]
		afterFilter := combineFilters(plan.BaseFilter, buildAdminRangeFilter(plan.Field, plan.Dir, cursorFromContent(sort, last), true))
		exists, err := store.ExistsAdminDoc(ctx, afterFilter)
		if err != nil {
			return nil, err
		}
		hasNext = exists
	} else {
		hasNext = hasMore
		if params.After != nil {
			first := contents[0]
			beforeFilter := combineFilters(plan.BaseFilter, buildAdminRangeFilter(plan.Field, plan.Dir, cursorFromContent(sort, first), false))
			exists, err := store.ExistsAdminDoc(ctx, beforeFilter)
			if err != nil {
				return nil, err
			}
			hasPrev = exists
		}
	}

	var nextCursor, prevCursor *string
	if hasNext {
		nc, err := EncodeAdminCursor(sort, contents[len(contents)-1])
		if err != nil {
			return nil, err
		}
		nextCursor = &nc
	}
	if hasPrev {
		pc, err := EncodeAdminCursor(sort, contents[0])
		if err != nil {
			return nil, err
		}
		prevCursor = &pc
	}

	return &domain.AdminContentPage{
		Contents:   contents,
		Total:      total,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}
