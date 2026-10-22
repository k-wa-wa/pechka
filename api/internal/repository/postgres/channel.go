package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type ChannelRepository struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{pool: pool}
}

func (r *ChannelRepository) List(ctx context.Context) ([]*domain.Channel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM channels
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Description, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, &ch)
	}
	return channels, rows.Err()
}

func (r *ChannelRepository) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	var ch domain.Channel
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM channels
		WHERE id = $1
	`, id).Scan(&ch.ID, &ch.Name, &ch.Description, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepository) Exists(ctx context.Context, id string) (bool, error) {
	var dummy int
	err := r.pool.QueryRow(ctx, `SELECT 1 FROM channels WHERE id = $1`, id).Scan(&dummy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
