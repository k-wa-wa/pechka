package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepositoryImpl struct {
	db *pgxpool.Pool
}

func (t *TransactionRepositoryImpl) Transaction(ctx context.Context, f func(tx *pgx.Tx) error) error {
	tx, err := t.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	err = f(&tx)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}
