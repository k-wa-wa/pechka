package mongo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestMongoRepositorySignatures(t *testing.T) {
	var contentRepo *ContentRepository
	var subRepo *SubtitleRepository

	if contentRepo != nil || subRepo != nil {
		t.Fatal("expected nil pointers")
	}

	params := ListParams{
		Limit:  10,
		Offset: 0,
	}
	if params.Limit != 10 {
		t.Errorf("unexpected limit: %d", params.Limit)
	}

	ctx := context.Background()
	if ctx == nil {
		t.Fatal("expected valid context")
	}
}

func TestAdminIndexModels(t *testing.T) {
	models := AdminIndexModels()
	if len(models) != 7 {
		t.Fatalf("expected 7 index models, got %d", len(models))
	}

	expected := []struct {
		name  string
		field string
		dir   int32
	}{
		{"idx_admin_v2_updated_desc", "updated_at", -1},
		{"idx_admin_v2_updated_asc", "updated_at", 1},
		{"idx_admin_v2_created_desc", "created_at", -1},
		{"idx_admin_v2_created_asc", "created_at", 1},
		{"idx_admin_v2_title_asc", "title", 1},
		{"idx_admin_v2_title_desc", "title", -1},
	}

	for i, exp := range expected {
		m := models[i]
		keys, ok := m.Keys.(bson.D)
		if !ok {
			t.Fatalf("model %d keys not bson.D", i)
		}
		wantKeys := bson.D{
			{Key: "deleted_at", Value: int32(1)},
			{Key: exp.field, Value: exp.dir},
			{Key: "_id", Value: int32(1)},
		}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("index %d keys = %v, want %v", i, keys, wantKeys)
		}
		if m.Options == nil {
			t.Fatalf("index %d options is nil", i)
		}
	}

	filterKeys, ok := models[6].Keys.(bson.D)
	if !ok {
		t.Fatalf("filter index keys not bson.D")
	}
	wantFilterKeys := bson.D{
		{Key: "deleted_at", Value: int32(1)},
		{Key: "status", Value: int32(1)},
		{Key: "channel_id", Value: int32(1)},
	}
	if !reflect.DeepEqual(filterKeys, wantFilterKeys) {
		t.Errorf("filter index keys = %v, want %v", filterKeys, wantFilterKeys)
	}
}

type mockMongo7IndexCatalog struct {
	indexes     []existingIndexSpec
	events      []string
	createCalls int
	dropCalls   int
	failCreate  error
	createHook  func(ctx context.Context, cat *mockMongo7IndexCatalog, models []mongo.IndexModel) error
}

func newMockMongo7IndexCatalog(initial []existingIndexSpec) *mockMongo7IndexCatalog {
	return &mockMongo7IndexCatalog{
		indexes: append([]existingIndexSpec(nil), initial...),
	}
}

func (m *mockMongo7IndexCatalog) List(_ context.Context) ([]existingIndexSpec, error) {
	m.events = append(m.events, "list")
	out := make([]existingIndexSpec, 0, len(m.indexes))
	for _, ex := range m.indexes {
		// MongoDB 7 の listIndexes (includeBuildUUIDs: true) 応答形式を BSON ラウンドトリップで再現する
		var rawDoc bson.M
		if ex.BuildUUID != nil {
			specDoc := bson.M{
				"v":    int32(2),
				"key":  ex.Key,
				"name": ex.Name,
			}
			if ex.Unique {
				specDoc["unique"] = true
			}
			if ex.Sparse {
				specDoc["sparse"] = true
			}
			if ex.Hidden {
				specDoc["hidden"] = true
			}
			rawDoc = bson.M{
				"spec":      specDoc,
				"buildUUID": ex.BuildUUID,
			}
		} else {
			rawDoc = bson.M{
				"v":    int32(2),
				"key":  ex.Key,
				"name": ex.Name,
			}
			if ex.Unique {
				rawDoc["unique"] = true
			}
			if ex.Sparse {
				rawDoc["sparse"] = true
			}
			if ex.Hidden {
				rawDoc["hidden"] = true
			}
		}
		b, err := bson.Marshal(rawDoc)
		if err != nil {
			return nil, err
		}
		var entry rawListIndexEntry
		if err := bson.Unmarshal(b, &entry); err != nil {
			return nil, err
		}
		out = append(out, entry.normalize())
	}
	return out, nil
}

func (m *mockMongo7IndexCatalog) CreateMany(ctx context.Context, models []mongo.IndexModel) ([]string, error) {
	m.createCalls++
	m.events = append(m.events, "create_many")
	if m.createHook != nil {
		if err := m.createHook(ctx, m, models); err != nil {
			return nil, err
		}
	}
	if m.failCreate != nil {
		return nil, m.failCreate
	}
	desiredNames := desiredAdminIndexNames()
	created := make([]string, 0, len(models))
	for _, model := range models {
		keys, _ := model.Keys.(bson.D)
		name := ""
		// AdminIndexModels の定義順と名前を照合してモック側の名前を解決する
		for i, am := range AdminIndexModels() {
			ak, _ := am.Keys.(bson.D)
			if indexKeysEqual(ak, keys) {
				name = desiredNames[i]
				break
			}
		}
		for i, ex := range m.indexes {
			if ex.Name == name && !hasSameKeysAndPlainOptions(ex, keys) {
				return nil, mongo.CommandError{Code: 86, Name: "IndexKeySpecsConflict", Message: "same name with different keys: " + name}
			}
			if ex.Name != name && hasSameKeysAndPlainOptions(ex, keys) {
				// MongoDB 7 は同一キー・同一オプションで名前だけ異なる作成を IndexOptionsConflict (85) で拒否する
				return nil, mongo.CommandError{Code: 85, Name: "IndexOptionsConflict", Message: "index already exists with a different name: " + ex.Name}
			}
			if ex.Name != name && indexKeysEqual(ex.Key, keys) && ex.Hidden {
				// 非表示インデックスが残存したまま別名で同一キーを作ろうとすると MongoDB 7 は IndexOptionsConflict (85) を返す
				return nil, mongo.CommandError{Code: 85, Name: "IndexOptionsConflict", Message: "hidden index exists with a different name: " + ex.Name}
			}
			if ex.Name == name && hasSameKeysAndPlainOptions(ex, keys) && ex.BuildUUID != nil {
				// 先行する構築中インデックスの完了を模倣して BuildUUID をクリアする
				m.indexes[i].BuildUUID = nil
			}
		}
		alreadySame := false
		for _, ex := range m.indexes {
			if ex.Name == name && hasSameKeysAndPlainOptions(ex, keys) {
				alreadySame = true
				break
			}
		}
		if !alreadySame {
			m.indexes = append(m.indexes, existingIndexSpec{Name: name, Key: keys})
		}
		created = append(created, name)
	}
	return created, nil
}

func (m *mockMongo7IndexCatalog) DropOne(_ context.Context, name string) error {
	m.dropCalls++
	m.events = append(m.events, "drop:"+name)
	for i, ex := range m.indexes {
		if ex.Name == name {
			m.indexes = append(m.indexes[:i], m.indexes[i+1:]...)
			return nil
		}
	}
	return mongo.CommandError{Code: 27, Name: "IndexNotFound", Message: "index not found: " + name}
}

func assertAll7AdminIndexesReady(t *testing.T, indexes []existingIndexSpec) {
	t.Helper()
	for _, model := range AdminIndexModels() {
		desiredKeys := model.Keys.(bson.D)
		found := false
		for _, ex := range indexes {
			if isReadyEquivalentPlainIndex(ex, desiredKeys) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing ready index for keys %v in %+v", desiredKeys, indexes)
		}
	}
}

func TestEnsureIndexesWithManager_Mongo7CompatibilityAndReentrancy(t *testing.T) {
	t.Run("fresh collection creates 7 v2 indexes and second init is a no-op", func(t *testing.T) {
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
		})
		repo1 := &ContentRepository{}
		if err := repo1.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("first init failed: %v", err)
		}
		if cat.createCalls != 1 || cat.dropCalls != 0 {
			t.Errorf("unexpected calls on fresh init: create=%d drop=%d", cat.createCalls, cat.dropCalls)
		}
		repo2 := &ContentRepository{}
		if err := repo2.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("second init failed: %v", err)
		}
		if cat.createCalls != 1 || cat.dropCalls != 0 {
			t.Errorf("second init should be no-op, got create=%d drop=%d", cat.createCalls, cat.dropCalls)
		}
	})

	t.Run("reuses previous 7 indexes with legacy names across two consecutive initializations without IndexOptionsConflict", func(t *testing.T) {
		// f8bd8da2 が作成していた「deleted_at 先頭付きだが旧名 idx_admin_*」の7本を再現
		prevModels := AdminIndexModels()
		initial := []existingIndexSpec{{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}}}
		for i, name := range legacyAdminIndexNames {
			initial = append(initial, existingIndexSpec{
				Name: name,
				Key:  prevModels[i].Keys.(bson.D),
			})
		}
		cat := newMockMongo7IndexCatalog(initial)

		repo1 := &ContentRepository{}
		if err := repo1.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("first init with existing 7 legacy-named indexes failed: %v", err)
		}
		repo2 := &ContentRepository{}
		if err := repo2.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("second consecutive init failed: %v", err)
		}
		if cat.createCalls != 0 {
			t.Errorf("expected 0 CreateMany calls when matching keys exist under legacy names, got %d", cat.createCalls)
		}
		if cat.dropCalls != 0 {
			t.Errorf("expected 0 DropOne calls for equivalent legacy-named indexes, got %d", cat.dropCalls)
		}
		if len(cat.indexes) != 8 { // _id_ + 7 reused
			t.Errorf("expected 8 indexes preserved, got %d", len(cat.indexes))
		}
	})

	t.Run("builds v2 indexes before dropping obsolete indexes without deleted_at", func(t *testing.T) {
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_updated_desc", Key: bson.D{{Key: "updated_at", Value: int32(-1)}, {Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_updated_asc", Key: bson.D{{Key: "updated_at", Value: int32(1)}, {Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_created_desc", Key: bson.D{{Key: "created_at", Value: int32(-1)}, {Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_created_asc", Key: bson.D{{Key: "created_at", Value: int32(1)}, {Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_title_asc", Key: bson.D{{Key: "title", Value: int32(1)}, {Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_title_desc", Key: bson.D{{Key: "title", Value: int32(-1)}, {Key: "_id", Value: int32(1)}}},
		})
		repo := &ContentRepository{}
		if err := repo.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("upgrade init failed: %v", err)
		}
		if len(cat.events) < 3 || cat.events[0] != "list" || cat.events[1] != "create_many" {
			t.Fatalf("expected list -> create_many -> drop order, got %v", cat.events)
		}
		if cat.dropCalls != 6 {
			t.Errorf("expected 6 obsolete legacy indexes dropped, got %d", cat.dropCalls)
		}
		for _, ex := range cat.indexes {
			if ex.Name != "_id_" && !containsStr(ex.Name, "v2") {
				t.Errorf("unexpected remaining non-v2 index %q", ex.Name)
			}
		}

		// Second init must be clean no-op
		repo2 := &ContentRepository{}
		if err := repo2.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("second init failed: %v", err)
		}
		if cat.createCalls != 1 || cat.dropCalls != 6 {
			t.Errorf("second init should not create or drop, create=%d drop=%d", cat.createCalls, cat.dropCalls)
		}
	})

	t.Run("does not reuse existing index with incompatible unique option", func(t *testing.T) {
		prevModels := AdminIndexModels()
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_updated_desc", Key: prevModels[0].Keys.(bson.D), Unique: true},
		})
		repo := &ContentRepository{}
		// Note: mock CreateMany rejects same-key plain vs unique if MongoDB wouldn't allow two same-key indexes;
		// here verify incompatible unique index is not treated as equivalent and CreateMany is attempted.
		cat.failCreate = errors.New("attempted create")
		err := repo.ensureIndexesWithManager(context.Background(), cat)
		if err == nil || !containsStr(err.Error(), "attempted create") {
			t.Fatalf("expected CreateMany to be attempted when existing index is unique, got %v", err)
		}
		if cat.dropCalls != 0 {
			t.Errorf("must not drop existing index when CreateMany fails, got dropCalls=%d", cat.dropCalls)
		}
	})

	t.Run("repairs mismatched v2-named index without losing ascending index that matched it", func(t *testing.T) {
		// idx_admin_v2_updated_desc という名前に昇順キー（updated_at: 1）が入っていた場合でも、
		// 撤去後に残る定義を基準に作成対象を決めるため、降順・昇順を含む7本すべてが完成する。
		ascKeys := AdminIndexModels()[1].Keys.(bson.D)
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_v2_updated_desc", Key: ascKeys},
		})
		repo := &ContentRepository{}
		if err := repo.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("repair init failed: %v", err)
		}
		assertAll7AdminIndexesReady(t, cat.indexes)
		if len(cat.indexes) != 8 { // _id_ + all 7 admin indexes
			t.Errorf("expected 8 total indexes, got %d (%+v)", len(cat.indexes), cat.indexes)
		}
	})

	t.Run("does not treat in-progress buildUUID index from another replica as complete", func(t *testing.T) {
		models := AdminIndexModels()
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
			{Name: "idx_admin_updated_desc", Key: bson.D{{Key: "updated_at", Value: int32(-1)}, {Key: "_id", Value: int32(1)}}},
			// 1台目のレプリカが構築中（buildUUID あり）の idx_admin_v2_updated_desc
			{Name: "idx_admin_v2_updated_desc", Key: models[0].Keys.(bson.D), BuildUUID: "building-uuid-1"},
		})

		// 1台目の構築が失敗したケース: 2台目も失敗し、旧インデックス撤去も indexesEnsured=true も行わない
		cat.createHook = func(_ context.Context, c *mockMongo7IndexCatalog, gotModels []mongo.IndexModel) error {
			if len(gotModels) != 7 {
				return errors.New("expected in-progress index to be included in CreateMany")
			}
			// 構築失敗により buildUUID 付きエントリは消滅
			c.indexes = c.indexes[:2]
			return errors.New("replica 1 index build failed")
		}
		repo2 := &ContentRepository{}
		err := repo2.ensureIndexesWithManager(context.Background(), cat)
		if err == nil || !containsStr(err.Error(), "replica 1 index build failed") {
			t.Fatalf("expected build failure error, got %v", err)
		}
		if repo2.indexesEnsured.Load() {
			t.Fatal("indexesEnsured must stay false when in-progress build fails")
		}
		if cat.dropCalls != 0 {
			t.Fatalf("must not drop legacy indexes before build succeeds, got dropCalls=%d", cat.dropCalls)
		}

		// 次のリクエストでは再検査して7本を構築し、旧定義を撤去して完了する
		cat.createHook = nil
		if err := repo2.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("retry after failed build should succeed: %v", err)
		}
		if !repo2.indexesEnsured.Load() {
			t.Fatal("expected indexesEnsured=true after retry succeeds")
		}
		assertAll7AdminIndexesReady(t, cat.indexes)
		if cat.dropCalls != 1 {
			t.Errorf("expected 1 obsolete legacy index dropped after build succeeds, got %d", cat.dropCalls)
		}
	})

	t.Run("does not reuse hidden legacy or v2 indexes and replaces them with visible indexes", func(t *testing.T) {
		models := AdminIndexModels()
		cat := newMockMongo7IndexCatalog([]existingIndexSpec{
			{Name: "_id_", Key: bson.D{{Key: "_id", Value: int32(1)}}},
			// 同一キーだが hidden: true の旧名インデックスと v2 名インデックス
			{Name: "idx_admin_updated_desc", Key: models[0].Keys.(bson.D), Hidden: true},
			{Name: "idx_admin_v2_created_desc", Key: models[2].Keys.(bson.D), Hidden: true},
		})
		repo := &ContentRepository{}
		if err := repo.ensureIndexesWithManager(context.Background(), cat); err != nil {
			t.Fatalf("ensureIndexesWithManager failed with hidden indexes: %v", err)
		}
		assertAll7AdminIndexesReady(t, cat.indexes)
		for _, ex := range cat.indexes {
			if ex.Hidden {
				t.Errorf("expected no hidden index remaining, found %+v", ex)
			}
		}
		if len(cat.indexes) != 8 {
			t.Errorf("expected 8 visible indexes (_id_ + 7 admin), got %d", len(cat.indexes))
		}
	})
}

func TestRawListIndexEntry_NormalizeMongo7SpecWrapper(t *testing.T) {
	// MongoDB 7 の listIndexes (includeBuildUUIDs: true) が返す実際の BSON 形式を検証
	rawDoc := bson.M{
		"spec": bson.M{
			"v":      int32(2),
			"key":    bson.D{{Key: "deleted_at", Value: int32(1)}, {Key: "updated_at", Value: int32(-1)}, {Key: "_id", Value: int32(1)}},
			"name":   "idx_admin_v2_updated_desc",
			"hidden": true,
		},
		"buildUUID": bson.Binary{Subtype: 4, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
	}
	b, err := bson.Marshal(rawDoc)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var entry rawListIndexEntry
	if err := bson.Unmarshal(b, &entry); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	norm := entry.normalize()
	if norm.Name != "idx_admin_v2_updated_desc" {
		t.Errorf("expected spec.name extracted, got %q", norm.Name)
	}
	if !norm.Hidden {
		t.Errorf("expected spec.hidden=true extracted")
	}
	if norm.BuildUUID == nil {
		t.Errorf("expected top-level buildUUID preserved")
	}
	wantKeys := AdminIndexModels()[0].Keys.(bson.D)
	if !indexKeysEqual(norm.Key, wantKeys) {
		t.Errorf("expected spec.key extracted, got %v", norm.Key)
	}
	if isReadyEquivalentPlainIndex(norm, wantKeys) {
		t.Errorf("in-progress/hidden spec must not be treated as ready equivalent index")
	}
}

func TestEncodeAndDecodeAdminCursor(t *testing.T) {
	ts := time.Date(2025, 5, 1, 12, 34, 56, 789456000, time.UTC)
	content := &domain.Content{
		ID:        "uuid-1",
		ShortID:   "short-1",
		Title:     "Alpha Title",
		CreatedAt: ts.Add(-time.Hour),
		UpdatedAt: ts,
	}

	for _, sort := range []domain.ContentSort{
		domain.ContentSortUpdatedDesc,
		domain.ContentSortUpdatedAsc,
		domain.ContentSortCreatedDesc,
		domain.ContentSortCreatedAsc,
	} {
		tok, err := EncodeAdminCursor(sort, content)
		if err != nil {
			t.Fatalf("encode %s: %v", sort, err)
		}
		cur, err := DecodeAdminCursor(tok, sort)
		if err != nil {
			t.Fatalf("decode %s: %v", sort, err)
		}
		if cur.Sort != sort || cur.ID != "short-1" {
			t.Errorf("unexpected cursor %+v", cur)
		}
		wantTime := ts
		if sort == domain.ContentSortCreatedDesc || sort == domain.ContentSortCreatedAsc {
			wantTime = ts.Add(-time.Hour)
		}
		// Millisecond precision preserved; sub-millisecond truncated
		if !cur.Time.Equal(wantTime.Truncate(time.Millisecond)) {
			t.Errorf("time mismatch for %s: got %v, want %v", sort, cur.Time, wantTime.Truncate(time.Millisecond))
		}
	}

	for _, sort := range []domain.ContentSort{domain.ContentSortTitleAsc, domain.ContentSortTitleDesc} {
		tok, err := EncodeAdminCursor(sort, content)
		if err != nil {
			t.Fatalf("encode %s: %v", sort, err)
		}
		cur, err := DecodeAdminCursor(tok, sort)
		if err != nil {
			t.Fatalf("decode %s: %v", sort, err)
		}
		if cur.Title != "Alpha Title" || cur.ID != "short-1" {
			t.Errorf("unexpected title cursor %+v", cur)
		}
	}

	// Empty title is still a valid string cursor
	emptyTitleTok, err := EncodeAdminCursor(domain.ContentSortTitleAsc, &domain.Content{ShortID: "s-empty", Title: ""})
	if err != nil {
		t.Fatalf("encode empty title: %v", err)
	}
	if cur, err := DecodeAdminCursor(emptyTitleTok, domain.ContentSortTitleAsc); err != nil || cur.Title != "" {
		t.Fatalf("expected valid empty title cursor, got %+v, err=%v", cur, err)
	}
}

func TestDecodeAdminCursor_Errors(t *testing.T) {
	validTok, _ := EncodeAdminCursor(domain.ContentSortUpdatedDesc, &domain.Content{
		ShortID:   "s1",
		UpdatedAt: time.Now().UTC(),
	})

	cases := []struct {
		name         string
		token        string
		expectedSort domain.ContentSort
	}{
		{"empty", "", domain.ContentSortUpdatedDesc},
		{"bad base64", "!!!not-base64!!!", domain.ContentSortUpdatedDesc},
		{"padded base64", validTok + "=", domain.ContentSortUpdatedDesc},
		{"sort mismatch", validTok, domain.ContentSortUpdatedAsc},
		{"invalid sort inside", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"bad","v":123,"id":"s1"}`)), ""},
		{"missing id", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"updated_desc","v":123}`)), domain.ContentSortUpdatedDesc},
		{"missing v", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"updated_desc","id":"s1"}`)), domain.ContentSortUpdatedDesc},
		{"null v", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"updated_desc","v":null,"id":"s1"}`)), domain.ContentSortUpdatedDesc},
		{"string v for time sort", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"updated_desc","v":"123","id":"s1"}`)), domain.ContentSortUpdatedDesc},
		{"float v for time sort", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"updated_desc","v":123.4,"id":"s1"}`)), domain.ContentSortUpdatedDesc},
		{"number v for title sort", base64.RawURLEncoding.EncodeToString([]byte(`{"sort":"title_asc","v":123,"id":"s1"}`)), domain.ContentSortTitleAsc},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeAdminCursor(tc.token, tc.expectedSort)
			if !errors.Is(err, ErrInvalidCursor) {
				t.Errorf("expected ErrInvalidCursor, got %v", err)
			}
		})
	}
}

func TestBuildAdminListQuery_AllSortsAndCursors(t *testing.T) {
	ts := time.UnixMilli(1700000000123).UTC()
	status := domain.ContentStatusReady
	ch := "bluray"

	sorts := []struct {
		sort  domain.ContentSort
		field string
		dir   int
	}{
		{domain.ContentSortUpdatedDesc, "updated_at", -1},
		{domain.ContentSortUpdatedAsc, "updated_at", 1},
		{domain.ContentSortCreatedDesc, "created_at", -1},
		{domain.ContentSortCreatedAsc, "created_at", 1},
		{domain.ContentSortTitleAsc, "title", 1},
		{domain.ContentSortTitleDesc, "title", -1},
	}

	for _, s := range sorts {
		// 1. No cursor
		plan, err := buildAdminListQuery(AdminListParams{
			Status:    &status,
			ChannelID: &ch,
			Sort:      s.sort,
			Limit:     15,
		})
		if err != nil {
			t.Fatalf("%s no cursor: %v", s.sort, err)
		}
		if plan.FetchLimit != 16 {
			t.Errorf("%s FetchLimit = %d, want 16", s.sort, plan.FetchLimit)
		}
		if _, hasArchived := plan.BaseFilter["archived_at"]; hasArchived {
			t.Errorf("%s must NOT exclude archived_at", s.sort)
		}
		if v, ok := plan.BaseFilter["deleted_at"]; !ok || v != nil {
			t.Errorf("%s must require deleted_at: nil", s.sort)
		}
		wantSort := bson.D{{Key: s.field, Value: s.dir}, {Key: "_id", Value: 1}}
		if !reflect.DeepEqual(plan.Sort, wantSort) {
			t.Errorf("%s sort = %v, want %v", s.sort, plan.Sort, wantSort)
		}

		cur := &AdminCursor{Sort: s.sort, Time: ts, Title: "Title", ID: "short-9"}

		// 2. After cursor
		afterPlan, err := buildAdminListQuery(AdminListParams{
			Sort:  s.sort,
			After: cur,
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("%s after: %v", s.sort, err)
		}
		if afterPlan.Reverse {
			t.Errorf("%s after should not be reverse", s.sort)
		}
		if !reflect.DeepEqual(afterPlan.Sort, wantSort) {
			t.Errorf("%s after sort = %v, want %v", s.sort, afterPlan.Sort, wantSort)
		}
		afterJSON, _ := json.Marshal(afterPlan.FindFilter)
		wantCmp := "$gt"
		if s.dir == -1 {
			wantCmp = "$lt"
		}
		if !containsStr(string(afterJSON), wantCmp) || !containsStr(string(afterJSON), `"_id":{"$gt":"short-9"}`) {
			t.Errorf("%s after filter unexpected: %s", s.sort, string(afterJSON))
		}

		// 3. Before cursor
		beforePlan, err := buildAdminListQuery(AdminListParams{
			Sort:   s.sort,
			Before: cur,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("%s before: %v", s.sort, err)
		}
		if !beforePlan.Reverse {
			t.Errorf("%s before should be reverse", s.sort)
		}
		wantRevSort := bson.D{{Key: s.field, Value: -s.dir}, {Key: "_id", Value: -1}}
		if !reflect.DeepEqual(beforePlan.Sort, wantRevSort) {
			t.Errorf("%s before sort = %v, want %v", s.sort, beforePlan.Sort, wantRevSort)
		}
		beforeJSON, _ := json.Marshal(beforePlan.FindFilter)
		wantRevCmp := "$lt"
		if s.dir == -1 {
			wantRevCmp = "$gt"
		}
		if !containsStr(string(beforeJSON), wantRevCmp) || !containsStr(string(beforeJSON), `"_id":{"$lt":"short-9"}`) {
			t.Errorf("%s before filter unexpected: %s", s.sort, string(beforeJSON))
		}
	}

	// Both after and before -> error
	cur := &AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: ts, ID: "s1"}
	if _, err := buildAdminListQuery(AdminListParams{
		Sort:   domain.ContentSortUpdatedDesc,
		After:  cur,
		Before: cur,
	}); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor when both after and before are set, got %v", err)
	}
}

func TestUnmigratedAdminDocFilterAndConversion(t *testing.T) {
	f := unmigratedAdminDocFilter()
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal filter: %v", err)
	}
	s := string(raw)
	for _, needle := range []string{`"id"`, `"created_at"`, `"updated_at"`, `"published_at"`, `"archived_at"`, `"deleted_at"`, `"$type":"date"`, `"$exists":false`} {
		if !containsStr(s, needle) {
			t.Errorf("unmigrated filter missing %s: %s", needle, s)
		}
	}

	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.FixedZone("JST", 9*3600))
	doc := mongoAdminDoc{
		ShortID:     "s-1",
		UUID:        "uuid-1",
		ContentType: domain.ContentTypeVideo,
		ChannelID:   "bluray",
		Title:       "Hello",
		Status:      domain.ContentStatusReady,
		Tags:        nil,
		ArchivedAt:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	c := doc.toDomain()
	if c.ID != "uuid-1" || c.ShortID != "s-1" {
		t.Errorf("unexpected IDs: %+v", c)
	}
	if c.Tags == nil || len(c.Tags) != 0 {
		t.Errorf("expected non-nil empty tags slice, got %#v", c.Tags)
	}
	if c.ArchivedAt == nil || c.ArchivedAt.Location() != time.UTC {
		t.Errorf("expected UTC ArchivedAt, got %v", c.ArchivedAt)
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

type mockAdminDocStore struct {
	findUnmigratedFn func(ctx context.Context, filter bson.M) (bool, error)
	countFn          func(ctx context.Context, filter bson.M) (int64, error)
	findDocsFn       func(ctx context.Context, filter bson.M, sort bson.D, limit int64) ([]mongoAdminDoc, error)
	existsFn         func(ctx context.Context, filter bson.M) (bool, error)
	existsCalls      int
}

func (m *mockAdminDocStore) FindUnmigrated(ctx context.Context, filter bson.M) (bool, error) {
	if m.findUnmigratedFn != nil {
		return m.findUnmigratedFn(ctx, filter)
	}
	return false, nil
}

func (m *mockAdminDocStore) CountDocuments(ctx context.Context, filter bson.M) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, filter)
	}
	return 0, nil
}

func (m *mockAdminDocStore) FindAdminDocs(ctx context.Context, filter bson.M, sort bson.D, limit int64) ([]mongoAdminDoc, error) {
	if m.findDocsFn != nil {
		return m.findDocsFn(ctx, filter, sort, limit)
	}
	return nil, nil
}

func (m *mockAdminDocStore) ExistsAdminDoc(ctx context.Context, filter bson.M) (bool, error) {
	m.existsCalls++
	if m.existsFn != nil {
		return m.existsFn(ctx, filter)
	}
	return false, nil
}

func makeMongoDoc(uuid, shortID, title string, ts time.Time) mongoAdminDoc {
	return mongoAdminDoc{
		ShortID:     shortID,
		UUID:        uuid,
		ContentType: domain.ContentTypeVideo,
		ChannelID:   "bluray",
		Title:       title,
		Status:      domain.ContentStatusReady,
		CreatedAt:   ts,
		UpdatedAt:   ts,
	}
}

func TestListAdminWithStore_PageAssemblyAndBoundaries(t *testing.T) {
	ctx := context.Background()
	tsSame := time.UnixMilli(1700000000123).UTC()

	t.Run("unmigrated document returns ErrUnmigratedReadModel", func(t *testing.T) {
		repo := &ContentRepository{}
		store := &mockAdminDocStore{
			findUnmigratedFn: func(_ context.Context, _ bson.M) (bool, error) { return true, nil },
		}
		_, err := repo.listAdminWithStore(ctx, store, AdminListParams{Limit: 2})
		if !errors.Is(err, ErrUnmigratedReadModel) {
			t.Fatalf("expected ErrUnmigratedReadModel, got %v", err)
		}
	})

	t.Run("0 items with cursor skips exists probe", func(t *testing.T) {
		repo := &ContentRepository{}
		cur := &AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSame, ID: "s1"}
		store := &mockAdminDocStore{
			countFn: func(_ context.Context, _ bson.M) (int64, error) { return 3, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, _ int64) ([]mongoAdminDoc, error) {
				return []mongoAdminDoc{}, nil
			},
		}
		page, err := repo.listAdminWithStore(ctx, store, AdminListParams{Sort: domain.ContentSortUpdatedDesc, After: cur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 0 || page.Total != 3 || page.HasNext || page.HasPrev || page.NextCursor != nil || page.PrevCursor != nil {
			t.Errorf("unexpected empty page: %+v", page)
		}
		if store.existsCalls != 0 {
			t.Errorf("expected 0 exists calls on empty result, got %d", store.existsCalls)
		}
	})

	t.Run("exact limit items hasNext is false", func(t *testing.T) {
		repo := &ContentRepository{}
		docs := []mongoAdminDoc{
			makeMongoDoc("u1", "s1", "Same", tsSame),
			makeMongoDoc("u2", "s2", "Same", tsSame),
		}
		store := &mockAdminDocStore{
			countFn: func(_ context.Context, _ bson.M) (int64, error) { return 2, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, limit int64) ([]mongoAdminDoc, error) {
				if limit != 3 {
					t.Errorf("expected fetch limit 3, got %d", limit)
				}
				return docs, nil
			},
		}
		page, err := repo.listAdminWithStore(ctx, store, AdminListParams{Sort: domain.ContentSortTitleAsc, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.HasNext || page.HasPrev || page.NextCursor != nil || page.PrevCursor != nil {
			t.Errorf("unexpected exact-limit page: %+v", page)
		}
	})

	t.Run("limit+1 items with after truncates and verifies tiebreaker cursors", func(t *testing.T) {
		repo := &ContentRepository{}
		afterCur := &AdminCursor{Sort: domain.ContentSortTitleAsc, Title: "Same", ID: "s0"}
		// Equal title boundary: s1, s2, s3 (limit=2 -> keep s1, s2; s3 triggers hasNext=true)
		docs := []mongoAdminDoc{
			makeMongoDoc("u1", "s1", "Same", tsSame),
			makeMongoDoc("u2", "s2", "Same", tsSame),
			makeMongoDoc("u3", "s3", "Same", tsSame),
		}
		var gotExistsFilter bson.M
		store := &mockAdminDocStore{
			countFn:    func(_ context.Context, _ bson.M) (int64, error) { return 4, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, _ int64) ([]mongoAdminDoc, error) { return docs, nil },
			existsFn: func(_ context.Context, filter bson.M) (bool, error) {
				gotExistsFilter = filter
				return true, nil
			},
		}
		page, err := repo.listAdminWithStore(ctx, store, AdminListParams{Sort: domain.ContentSortTitleAsc, After: afterCur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.Contents[0].ShortID != "s1" || page.Contents[1].ShortID != "s2" {
			t.Fatalf("unexpected truncated contents: %+v", page.Contents)
		}
		if !page.HasNext || !page.HasPrev || page.NextCursor == nil || page.PrevCursor == nil {
			t.Fatalf("expected both cursors and flags true, got %+v", page)
		}
		nc, err := DecodeAdminCursor(*page.NextCursor, domain.ContentSortTitleAsc)
		if err != nil || nc.ID != "s2" || nc.Title != "Same" {
			t.Errorf("unexpected next cursor: %+v err=%v", nc, err)
		}
		pc, err := DecodeAdminCursor(*page.PrevCursor, domain.ContentSortTitleAsc)
		if err != nil || pc.ID != "s1" || pc.Title != "Same" {
			t.Errorf("unexpected prev cursor: %+v err=%v", pc, err)
		}
		rawExists, _ := json.Marshal(gotExistsFilter)
		if !containsStr(string(rawExists), `"_id":{"$lt":"s1"}`) {
			t.Errorf("expected prev exists filter to probe before s1, got %s", string(rawExists))
		}
	})

	t.Run("before cursor reverses slice back to forward order and checks hasNext", func(t *testing.T) {
		repo := &ContentRepository{}
		beforeCur := &AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSame, ID: "s4"}
		// Reverse query returns [s3, s2, s1]; limit=2 -> slice [:2]=[s3, s2], reverse -> [s2, s3], hasPrev=true
		docsDescReverse := []mongoAdminDoc{
			makeMongoDoc("u3", "s3", "T3", tsSame),
			makeMongoDoc("u2", "s2", "T2", tsSame),
			makeMongoDoc("u1", "s1", "T1", tsSame),
		}
		var gotExistsFilter bson.M
		store := &mockAdminDocStore{
			countFn: func(_ context.Context, _ bson.M) (int64, error) { return 4, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, _ int64) ([]mongoAdminDoc, error) {
				return docsDescReverse, nil
			},
			existsFn: func(_ context.Context, filter bson.M) (bool, error) {
				gotExistsFilter = filter
				return true, nil
			},
		}
		page, err := repo.listAdminWithStore(ctx, store, AdminListParams{Sort: domain.ContentSortUpdatedDesc, Before: beforeCur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.Contents[0].ShortID != "s2" || page.Contents[1].ShortID != "s3" {
			t.Fatalf("expected reversed forward order [s2, s3], got %+v", page.Contents)
		}
		if !page.HasPrev || !page.HasNext || page.PrevCursor == nil || page.NextCursor == nil {
			t.Fatalf("expected hasPrev & hasNext true, got %+v", page)
		}
		pc, _ := DecodeAdminCursor(*page.PrevCursor, domain.ContentSortUpdatedDesc)
		nc, _ := DecodeAdminCursor(*page.NextCursor, domain.ContentSortUpdatedDesc)
		if pc.ID != "s2" || nc.ID != "s3" {
			t.Errorf("unexpected cursors after reverse: prev=%s next=%s", pc.ID, nc.ID)
		}
		rawExists, _ := json.Marshal(gotExistsFilter)
		if !containsStr(string(rawExists), `"_id":{"$gt":"s3"}`) {
			t.Errorf("expected next exists filter to probe after s3, got %s", string(rawExists))
		}
	})

	t.Run("exists probe error is propagated", func(t *testing.T) {
		repo := &ContentRepository{}
		afterCur := &AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSame, ID: "s0"}
		store := &mockAdminDocStore{
			countFn: func(_ context.Context, _ bson.M) (int64, error) { return 2, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, _ int64) ([]mongoAdminDoc, error) {
				return []mongoAdminDoc{makeMongoDoc("u1", "s1", "T1", tsSame)}, nil
			},
			existsFn: func(_ context.Context, _ bson.M) (bool, error) {
				return false, errors.New("mongo read timeout")
			},
		}
		_, err := repo.listAdminWithStore(ctx, store, AdminListParams{Sort: domain.ContentSortUpdatedDesc, After: afterCur, Limit: 2})
		if err == nil || !containsStr(err.Error(), "mongo read timeout") {
			t.Fatalf("expected exists error to propagate, got %v", err)
		}
	})
}

func TestContentRepository_MigrationCheckTimeoutAndStartupCheck(t *testing.T) {
	t.Run("independent migration check timeout completes slow scan despite short request deadline", func(t *testing.T) {
		repo := (&ContentRepository{}).WithMigrationCheckTimeout(500 * time.Millisecond)
		var checkCalls int
		store := &mockAdminDocStore{
			findUnmigratedFn: func(ctx context.Context, _ bson.M) (bool, error) {
				checkCalls++
				select {
				case <-time.After(80 * time.Millisecond):
					return false, nil
				case <-ctx.Done():
					return false, ctx.Err()
				}
			},
			countFn: func(_ context.Context, _ bson.M) (int64, error) { return 0, nil },
			findDocsFn: func(_ context.Context, _ bson.M, _ bson.D, _ int64) ([]mongoAdminDoc, error) {
				return nil, nil
			},
		}

		// ブレーカー風の短い期限（20ms）でも未移行検査は独立期限（500ms）で完了し migrated=true になる
		reqCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		page, err := repo.listAdminWithStore(reqCtx, store, AdminListParams{Limit: 10})
		if err != nil {
			t.Fatalf("expected migration check and list to succeed, got %v", err)
		}
		if len(page.Contents) != 0 || checkCalls != 1 || !repo.migrated.Load() {
			t.Fatalf("unexpected state: page=%+v checkCalls=%d migrated=%v", page, checkCalls, repo.migrated.Load())
		}

		// 2回目は FindUnmigrated を再実行しない
		_, err = repo.listAdminWithStore(context.Background(), store, AdminListParams{Limit: 10})
		if err != nil {
			t.Fatalf("second call failed: %v", err)
		}
		if checkCalls != 1 {
			t.Errorf("expected FindUnmigrated to run once, ran %d times", checkCalls)
		}
	})

	t.Run("checkMigratedWithStore keeps migrated=false when unmigrated document exists", func(t *testing.T) {
		repo := &ContentRepository{}
		store := &mockAdminDocStore{
			findUnmigratedFn: func(_ context.Context, _ bson.M) (bool, error) { return true, nil },
		}
		err := repo.checkMigratedWithStore(context.Background(), store)
		if !errors.Is(err, ErrUnmigratedReadModel) {
			t.Fatalf("expected ErrUnmigratedReadModel, got %v", err)
		}
		if repo.migrated.Load() {
			t.Fatal("migrated must stay false when unmigrated doc exists")
		}
	})
}
