//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestContentRepository_SetArchived_TwoConnectionConcurrency(t *testing.T) {
	dsn := os.Getenv("PECHKA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Fatal("PECHKA_TEST_POSTGRES_DSN is required for integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}
	defer pool.Close()

	repo := NewContentRepository(pool)

	// 1. 同時 Archive: 2接続で同一行を同時にアーカイブし、両方の応答とDB保存状態が完全一致することを検証
	created, err := repo.Create(ctx, CreateContentParams{
		ShortID:     "conc-arch-1",
		ContentType: domain.ContentTypeVideo,
		ChannelID:   "bluray",
		Title:       "Concurrent Archive",
		Status:      domain.ContentStatusReady,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 先行トランザクションで明示的に行ロックして競合タイミングを固定する
	conn1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire conn1 failed: %v", err)
	}
	defer conn1.Release()

	tx1, err := conn1.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	firstUpdated, err := scanContent(tx1.QueryRow(ctx, updateArchivedTrueQuery, created.ID))
	if err != nil {
		_ = tx1.Rollback(ctx)
		t.Fatalf("tx1 updateArchivedTrueQuery failed: %v", err)
	}

	var wg sync.WaitGroup
	var res2 *domain.Content
	var err2 error
	wg.Add(1)
	go func() {
		defer wg.Done()
		res2, err2 = repo.SetArchived(ctx, created.ID, true)
	}()

	time.Sleep(80 * time.Millisecond)
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("tx1 commit failed: %v", err)
	}
	wg.Wait()

	if err2 != nil {
		t.Fatalf("concurrent SetArchived(true) failed: %v", err2)
	}
	if res2.ArchivedAt == nil || !res2.ArchivedAt.Equal(*firstUpdated.ArchivedAt) || !res2.UpdatedAt.Equal(firstUpdated.UpdatedAt) {
		t.Fatalf("concurrent Archive mismatch: got (%v, %v), want (%v, %v)", res2.ArchivedAt, res2.UpdatedAt, firstUpdated.ArchivedAt, firstUpdated.UpdatedAt)
	}
	stored, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if stored.ArchivedAt == nil || !stored.ArchivedAt.Equal(*res2.ArchivedAt) || !stored.UpdatedAt.Equal(res2.UpdatedAt) {
		t.Fatalf("stored row mismatch after concurrent Archive: stored=%+v res2=%+v", stored, res2)
	}

	// 2. 同時 Delete + Unarchive: 削除のコミットを待った後の復元が削除前を返さず pgx.ErrNoRows になることを検証
	txDel, err := conn1.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("BeginTx delete failed: %v", err)
	}
	if _, err := txDel.Exec(ctx, deleteContentQuery, created.ID); err != nil {
		_ = txDel.Rollback(ctx)
		t.Fatalf("txDel Exec failed: %v", err)
	}

	var errUnarch error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errUnarch = repo.SetArchived(ctx, created.ID, false)
	}()

	time.Sleep(80 * time.Millisecond)
	if err := txDel.Commit(ctx); err != nil {
		t.Fatalf("txDel commit failed: %v", err)
	}
	wg.Wait()

	if !errors.Is(errUnarch, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows when Delete commits before SetArchived(false), got %v", errUnarch)
	}
}
