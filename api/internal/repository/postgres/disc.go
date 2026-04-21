package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type DiscRepository struct {
	pool *pgxpool.Pool
}

func NewDiscRepository(pool *pgxpool.Pool) *DiscRepository {
	return &DiscRepository{pool: pool}
}

type CreateDiscParams struct {
	Label    string
	DiscName *string
}

func (r *DiscRepository) Create(ctx context.Context, params CreateDiscParams) (*domain.Disc, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO discs (label, disc_name) VALUES ($1, $2)
		RETURNING id, label, disc_name, created_at
	`, params.Label, params.DiscName)

	return scanDisc(row)
}

func (r *DiscRepository) List(ctx context.Context) ([]*domain.Disc, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, label, disc_name, created_at FROM discs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var discs []*domain.Disc
	for rows.Next() {
		d, err := scanDisc(rows)
		if err != nil {
			return nil, err
		}
		discs = append(discs, d)
	}
	return discs, rows.Err()
}

func scanDisc(row scanner) (*domain.Disc, error) {
	var d domain.Disc
	err := row.Scan(&d.ID, &d.Label, &d.DiscName, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
