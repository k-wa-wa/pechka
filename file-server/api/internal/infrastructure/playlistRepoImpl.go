package infrastructure

import (
	"context"
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

type playlistRepoImpl struct {
	db *pgxpool.Pool
}

func NewPlaylistRepo(db *pgxpool.Pool) repository.PlaylistRepo {
	return &playlistRepoImpl{db: db}
}

func (pri *playlistRepoImpl) Select(playlistId string) (*model.PlaylistEntity, error) {
	query := `
		SELECT
			id, title, description, created_at, updated_at
		FROM
			playlists
		WHERE
			id = $1
	`

	var playlist model.PlaylistEntity
	if err := pri.db.QueryRow(context.Background(), query, playlistId).Scan(
		&playlist.Id,
		&playlist.Title,
		&playlist.Description,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &playlist, nil
}

func (pri *playlistRepoImpl) SelectVideos(playlistId string, limit, offset int) ([]*model.VideoEntity, error) {
	query := `
		SELECT
			id, fullpath, title, description, url, created_at, updated_at
		FROM videos v
		INNER JOIN playlist_videos pv
			ON pv.video_id = v.id
		WHERE
			pv.playlist_id = $1
		ORDER BY
			pv.position DESC, v.updated_at DESC
		LIMIT $2
		OFFSET $3
	`

	log.Info(time.Now())
	rows, err := pri.db.Query(context.Background(), query, playlistId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []*model.VideoEntity{}
	for rows.Next() {
		var videoEntity model.VideoEntity
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
	log.Info(time.Now())

	return videos, nil
}

func (pri *playlistRepoImpl) CountVideos(playlistId string) (int, error) {
	query := `SELECT COUNT(1) FROM playlist_videos WHERE playlist_id = $1`

	var count int
	if err := pri.db.QueryRow(context.Background(), query, playlistId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
