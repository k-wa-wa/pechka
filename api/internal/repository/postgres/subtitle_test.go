package postgres

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestSubtitleQueriesExcludeDeletedContents(t *testing.T) {
	queries := map[string]string{
		"listTracksQuery":           listTracksQuery,
		"getTrackQuery":             getTrackQuery,
		"lockParentByTrackQuery":    lockParentByTrackQuery,
		"listCuesQuery":             listCuesQuery,
		"lockParentByCueQuery":      lockParentByCueQuery,
		"insertCueCheckParentQuery": insertCueCheckParentQuery,
	}
	for name, q := range queries {
		if !strings.Contains(q, "c.deleted_at IS NULL") {
			t.Errorf("%s must filter c.deleted_at IS NULL, got %q", name, q)
		}
	}
	if !strings.Contains(lockParentByTrackQuery, "FOR SHARE OF c") {
		t.Errorf("lockParentByTrackQuery must lock only parent c FOR SHARE OF c: %q", lockParentByTrackQuery)
	}
	if !strings.Contains(lockParentByCueQuery, "FOR SHARE OF c") {
		t.Errorf("lockParentByCueQuery must lock only parent c FOR SHARE OF c: %q", lockParentByCueQuery)
	}

	text := "updated"
	qUpdate, args := buildUpdateCueQuery(UpdateCueParams{ID: "cue-1", Text: &text})
	if len(args) != 2 || args[0] != "updated" || args[1] != "cue-1" {
		t.Errorf("unexpected update cue args: %v", args)
	}
	if !strings.Contains(qUpdate, "UPDATE subtitle_cues SET") {
		t.Errorf("unexpected update cue query: %q", qUpdate)
	}
}

type fakeSubtitleRow struct {
	scanFn func(dest ...any) error
}

func (r fakeSubtitleRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

type inMemoryLockDB struct {
	mu           sync.Mutex
	parentLocked bool
	parentCond   *sync.Cond
	deleted      bool
	trackStatus  domain.SubtitleTrackStatus
	cueText      string
	cueDeleted   bool
	// PostgreSQL の FOR SHARE（対象指定なし）による st / cu 共有ロック競合・デッドロックを再現するための状態
	trackShareHolders int
	cueShareHolders   int
	lockBarrier       *sync.WaitGroup
}

func newInMemoryLockDB() *inMemoryLockDB {
	db := &inMemoryLockDB{
		trackStatus: domain.SubtitleTrackStatusDraft,
		cueText:     "initial",
	}
	db.parentCond = sync.NewCond(&db.mu)
	return db
}

type mockSubtitleTx struct {
	db             *inMemoryLockDB
	events         *[]string
	heldTrackShare bool
	heldCueShare   bool
}

func (tx *mockSubtitleTx) releaseChildShares() {
	tx.db.mu.Lock()
	if tx.heldTrackShare {
		tx.db.trackShareHolders--
		tx.heldTrackShare = false
	}
	if tx.heldCueShare {
		tx.db.cueShareHolders--
		tx.heldCueShare = false
	}
	tx.db.mu.Unlock()
}

func (tx *mockSubtitleTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "FOR SHARE") {
		*tx.events = append(*tx.events, "lock_parent")
		locksChildTables := !strings.Contains(sql, "FOR SHARE OF c")
		locksTrack := locksChildTables && strings.Contains(sql, "subtitle_tracks st")
		locksCue := locksChildTables && strings.Contains(sql, "subtitle_cues cu")
		return fakeSubtitleRow{scanFn: func(dest ...any) error {
			tx.db.mu.Lock()
			for tx.db.parentLocked {
				tx.db.parentCond.Wait()
			}
			deleted := tx.db.deleted
			if !deleted {
				if locksTrack {
					tx.db.trackShareHolders++
					tx.heldTrackShare = true
				}
				if locksCue {
					tx.db.cueShareHolders++
					tx.heldCueShare = true
				}
			}
			barrier := tx.db.lockBarrier
			tx.db.mu.Unlock()
			if deleted {
				return pgx.ErrNoRows
			}
			if barrier != nil {
				barrier.Done()
				barrier.Wait()
			}
			*(dest[0].(*int)) = 1
			return nil
		}}
	}
	if strings.HasPrefix(sql, "UPDATE subtitle_tracks") {
		*tx.events = append(*tx.events, "update_track")
		return fakeSubtitleRow{scanFn: func(dest ...any) error {
			now := time.Now().UTC()
			tx.db.mu.Lock()
			otherHolders := tx.db.trackShareHolders
			if tx.heldTrackShare {
				otherHolders--
			}
			if otherHolders > 0 {
				tx.db.mu.Unlock()
				return errors.New("deadlock detected while upgrading track SHARE lock to UPDATE lock")
			}
			tx.db.trackStatus = domain.SubtitleTrackStatusPublished
			tx.db.mu.Unlock()
			*(dest[0].(*string)) = "t1"
			*(dest[1].(*string)) = "c1"
			*(dest[2].(*string)) = "ja"
			*(dest[3].(*domain.SubtitleTrackStatus)) = domain.SubtitleTrackStatusPublished
			*(dest[4].(*string)) = "large-v3"
			*(dest[5].(*time.Time)) = now
			*(dest[6].(*time.Time)) = now
			return nil
		}}
	}
	if strings.HasPrefix(sql, "UPDATE subtitle_cues") {
		*tx.events = append(*tx.events, "update_cue")
		return fakeSubtitleRow{scanFn: func(dest ...any) error {
			now := time.Now().UTC()
			tx.db.mu.Lock()
			otherHolders := tx.db.cueShareHolders
			if tx.heldCueShare {
				otherHolders--
			}
			if otherHolders > 0 {
				tx.db.mu.Unlock()
				return errors.New("deadlock detected while upgrading cue SHARE lock to UPDATE lock")
			}
			tx.db.cueText = "edited"
			tx.db.mu.Unlock()
			*(dest[0].(*string)) = "cue1"
			*(dest[1].(*string)) = "t1"
			*(dest[2].(*int)) = 1
			*(dest[3].(*int)) = 0
			*(dest[4].(*int)) = 1000
			*(dest[5].(*string)) = "edited"
			*(dest[6].(*string)) = "initial"
			*(dest[7].(*bool)) = false
			*(dest[8].(*time.Time)) = now
			return nil
		}}
	}
	return fakeSubtitleRow{scanFn: func(_ ...any) error { return errors.New("unexpected query") }}
}

func (tx *mockSubtitleTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if strings.HasPrefix(sql, "DELETE FROM subtitle_cues") {
		*tx.events = append(*tx.events, "delete_cue")
		tx.db.mu.Lock()
		tx.db.cueDeleted = true
		tx.db.mu.Unlock()
		return pgconn.NewCommandTag("DELETE 1"), nil
	}
	return pgconn.CommandTag{}, errors.New("unexpected exec")
}

func (tx *mockSubtitleTx) Commit(_ context.Context) error {
	*tx.events = append(*tx.events, "commit")
	tx.releaseChildShares()
	return nil
}

func (tx *mockSubtitleTx) Rollback(_ context.Context) error {
	*tx.events = append(*tx.events, "rollback")
	tx.releaseChildShares()
	return nil
}

type mockSubtitleBeginner struct {
	db     *inMemoryLockDB
	events *[]string
}

func (b *mockSubtitleBeginner) Begin(_ context.Context) (subtitleTx, error) {
	*b.events = append(*b.events, "begin")
	return &mockSubtitleTx{db: b.db, events: b.events}, nil
}

func TestSubtitleMutations_ConcurrentParentDeleteRejected(t *testing.T) {
	t.Run("SetTrackStatus waits for concurrent delete and returns ErrNoRows", func(t *testing.T) {
		db := newInMemoryLockDB()
		db.parentLocked = true
		events := []string{}
		beginner := &mockSubtitleBeginner{db: db, events: &events}

		var wg sync.WaitGroup
		var gotErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, gotErr = setTrackStatusWithBeginner(context.Background(), beginner, "t1", domain.SubtitleTrackStatusPublished)
		}()

		time.Sleep(20 * time.Millisecond)
		db.mu.Lock()
		db.deleted = true
		db.parentLocked = false
		db.parentCond.Broadcast()
		db.mu.Unlock()

		wg.Wait()
		if !errors.Is(gotErr, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows after concurrent parent delete, got %v", gotErr)
		}
		if db.trackStatus != domain.SubtitleTrackStatusDraft {
			t.Errorf("track status must remain draft, got %q", db.trackStatus)
		}
		if got := strings.Join(events, ","); got != "begin,lock_parent,rollback" {
			t.Errorf("unexpected execution order: %q", got)
		}
	})

	t.Run("UpdateCue waits for concurrent delete and returns ErrNoRows", func(t *testing.T) {
		db := newInMemoryLockDB()
		db.parentLocked = true
		events := []string{}
		beginner := &mockSubtitleBeginner{db: db, events: &events}
		edited := "edited"

		var wg sync.WaitGroup
		var gotErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, gotErr = updateCueWithBeginner(context.Background(), beginner, UpdateCueParams{ID: "cue1", Text: &edited})
		}()

		time.Sleep(20 * time.Millisecond)
		db.mu.Lock()
		db.deleted = true
		db.parentLocked = false
		db.parentCond.Broadcast()
		db.mu.Unlock()

		wg.Wait()
		if !errors.Is(gotErr, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows after concurrent parent delete, got %v", gotErr)
		}
		if db.cueText != "initial" {
			t.Errorf("cue text must remain unchanged, got %q", db.cueText)
		}
		if got := strings.Join(events, ","); got != "begin,lock_parent,rollback" {
			t.Errorf("unexpected execution order: %q", got)
		}
	})

	t.Run("DeleteCue waits for concurrent delete and returns ErrNoRows", func(t *testing.T) {
		db := newInMemoryLockDB()
		db.parentLocked = true
		events := []string{}
		beginner := &mockSubtitleBeginner{db: db, events: &events}

		var wg sync.WaitGroup
		var gotErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			gotErr = deleteCueWithBeginner(context.Background(), beginner, "cue1")
		}()

		time.Sleep(20 * time.Millisecond)
		db.mu.Lock()
		db.deleted = true
		db.parentLocked = false
		db.parentCond.Broadcast()
		db.mu.Unlock()

		wg.Wait()
		if !errors.Is(gotErr, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows after concurrent parent delete, got %v", gotErr)
		}
		if db.cueDeleted {
			t.Errorf("cue must not be deleted when parent delete commits first")
		}
		if got := strings.Join(events, ","); got != "begin,lock_parent,rollback" {
			t.Errorf("unexpected execution order: %q", got)
		}
	})

	t.Run("Mutations succeed when parent remains undeleted", func(t *testing.T) {
		db := newInMemoryLockDB()
		events := []string{}
		beginner := &mockSubtitleBeginner{db: db, events: &events}
		edited := "edited"

		if tr, err := setTrackStatusWithBeginner(context.Background(), beginner, "t1", domain.SubtitleTrackStatusPublished); err != nil || tr.Status != domain.SubtitleTrackStatusPublished {
			t.Fatalf("SetTrackStatus failed: %v %+v", err, tr)
		}
		if cue, err := updateCueWithBeginner(context.Background(), beginner, UpdateCueParams{ID: "cue1", Text: &edited}); err != nil || cue.Text != "edited" {
			t.Fatalf("UpdateCue failed: %v %+v", err, cue)
		}
		if err := deleteCueWithBeginner(context.Background(), beginner, "cue1"); err != nil || !db.cueDeleted {
			t.Fatalf("DeleteCue failed: %v deleted=%v", err, db.cueDeleted)
		}
	})
}

func TestSubtitleMutations_ConcurrentSameTargetNoDeadlock(t *testing.T) {
	t.Run("concurrent SetTrackStatus on same track avoids SHARE lock deadlock", func(t *testing.T) {
		db := newInMemoryLockDB()
		var barrier sync.WaitGroup
		barrier.Add(2)
		db.lockBarrier = &barrier

		var mu sync.Mutex
		eventsA, eventsB := []string{}, []string{}
		beginnerA := &mockSubtitleBeginner{db: db, events: &eventsA}
		beginnerB := &mockSubtitleBeginner{db: db, events: &eventsB}

		var wg sync.WaitGroup
		var errA, errB error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, errA = setTrackStatusWithBeginner(context.Background(), beginnerA, "t1", domain.SubtitleTrackStatusPublished)
		}()
		go func() {
			defer wg.Done()
			_, errB = setTrackStatusWithBeginner(context.Background(), beginnerB, "t1", domain.SubtitleTrackStatusPublished)
		}()
		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		if errA != nil || errB != nil {
			t.Fatalf("expected no deadlock errors, got errA=%v errB=%v", errA, errB)
		}
	})

	t.Run("concurrent UpdateCue on same cue avoids SHARE lock deadlock", func(t *testing.T) {
		db := newInMemoryLockDB()
		var barrier sync.WaitGroup
		barrier.Add(2)
		db.lockBarrier = &barrier

		eventsA, eventsB := []string{}, []string{}
		beginnerA := &mockSubtitleBeginner{db: db, events: &eventsA}
		beginnerB := &mockSubtitleBeginner{db: db, events: &eventsB}
		edited := "edited"

		var wg sync.WaitGroup
		var errA, errB error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, errA = updateCueWithBeginner(context.Background(), beginnerA, UpdateCueParams{ID: "cue1", Text: &edited})
		}()
		go func() {
			defer wg.Done()
			_, errB = updateCueWithBeginner(context.Background(), beginnerB, UpdateCueParams{ID: "cue1", Text: &edited})
		}()
		wg.Wait()

		if errA != nil || errB != nil {
			t.Fatalf("expected no deadlock errors, got errA=%v errB=%v", errA, errB)
		}
	})
}
