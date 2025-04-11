package infrastructure

import (
	"context"
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type videoRepoImpl struct {
	db *pgxpool.Pool
}

func NewVideoRepo(db *pgxpool.Pool) repository.VideoRepo {
	return &videoRepoImpl{db: db}
}

func (vri *videoRepoImpl) SelectOne(id string) (*model.VideoEntity, error) {
	var videoEntity model.VideoEntity
	if err := vri.db.QueryRow(context.Background(), `select * from videos where id = $1`, id).Scan(
		&videoEntity.Id,
		&videoEntity.Fullpath,
		&videoEntity.Title,
		&videoEntity.Description,
		&videoEntity.Url,
		&videoEntity.CreatedAt,
		&videoEntity.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &videoEntity, nil
}

func (vri *videoRepoImpl) Update(id, title, description string) (*model.VideoEntity, error) {
	var videoEntity model.VideoEntity
	if err := vri.db.QueryRow(
		context.Background(),
		`UPDATE videos SET title = $1, description = $2 where id = $3 RETURNING *`, title, description, id).Scan(
		&videoEntity.Id,
		&videoEntity.Fullpath,
		&videoEntity.Title,
		&videoEntity.Description,
		&videoEntity.Url,
		&videoEntity.CreatedAt,
		&videoEntity.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &videoEntity, nil
}
