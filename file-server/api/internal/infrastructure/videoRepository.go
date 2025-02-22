package infrastructure

import (
	"context"
	"pechka/file-server/internal/db"
	"time"

	"github.com/gofiber/fiber/v2/log"
)

type VideoEntity struct {
	Id          string     `json:"id"`
	Fullpath    string     `json:"-"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Url         string     `json:"url"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}

type VideoRepo interface {
	Select(playlistId, fromId string, limit int) ([]*VideoEntity, error)
	SelectOne(id string) (*VideoEntity, error)
	Update(id, title, description string) (*VideoEntity, error)
}

type VideoRepoImpl struct {
	Db db.DB
}

/* playlistIdは未実装 */
func (vri *VideoRepoImpl) Select(playlistId, fromId string, limit int) ([]*VideoEntity, error) {
	query := `
		WITH from_video AS (
			SELECT id, updated_at
			FROM videos
			WHERE id = $1
		)
		SELECT *
		FROM videos
		WHERE
			NOT EXISTS (SELECT 1 FROM from_video)
			OR (
				updated_at < (SELECT updated_at FROM from_video)
				OR (
					updated_at = (SELECT updated_at FROM from_video)
					AND id >= (SELECT id FROM from_video)
				)
			)
		ORDER BY updated_at DESC, id ASC
		LIMIT $2
	`

	rows, err := vri.Db.Query(context.Background(), query, fromId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []*VideoEntity{}
	for rows.Next() {
		var videoEntity VideoEntity
		if err := rows.Scan(
			&videoEntity.Id,
			&videoEntity.Fullpath,
			&videoEntity.Title,
			&videoEntity.Description,
			&videoEntity.Url,
			&videoEntity.CreatedAt,
			&videoEntity.UpdatedAt,
		); err != nil {
			log.Warn(err)
		}
		videos = append(videos, &videoEntity)
	}

	return videos, nil
}

func (vri *VideoRepoImpl) SelectOne(id string) (*VideoEntity, error) {
	var videoEntity VideoEntity
	if err := vri.Db.QueryRow(context.Background(), `select * from videos where id = $1`, id).Scan(
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

func (vri *VideoRepoImpl) Update(id, title, description string) (*VideoEntity, error) {
	var videoEntity VideoEntity
	if err := vri.Db.QueryRow(
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
