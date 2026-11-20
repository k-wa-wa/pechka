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
	if len(models) != 6 {
		t.Fatalf("expected 6 index models, got %d", len(models))
	}

	expected := []struct {
		name  string
		field string
		dir   int32
	}{
		{"idx_admin_updated_desc", "updated_at", -1},
		{"idx_admin_updated_asc", "updated_at", 1},
		{"idx_admin_created_desc", "created_at", -1},
		{"idx_admin_created_asc", "created_at", 1},
		{"idx_admin_title_asc", "title", 1},
		{"idx_admin_title_desc", "title", -1},
	}

	for i, exp := range expected {
		m := models[i]
		keys, ok := m.Keys.(bson.D)
		if !ok {
			t.Fatalf("model %d keys not bson.D", i)
		}
		wantKeys := bson.D{{Key: exp.field, Value: exp.dir}, {Key: "_id", Value: int32(1)}}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("index %d keys = %v, want %v", i, keys, wantKeys)
		}
		if m.Options == nil {
			t.Fatalf("index %d options is nil", i)
		}
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
