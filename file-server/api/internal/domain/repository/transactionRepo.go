package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TransactionRepo interface {
	Transaction(ctx context.Context, f func(tx *pgx.Tx) error) error
}
