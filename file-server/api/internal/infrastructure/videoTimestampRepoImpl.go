package infrastructure

import (
	"context"
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"

	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type videoTimestampRepoImpl struct {
	db *pgxpool.Pool
}

func NewVideoTimestampRepo(db *pgxpool.Pool) repository.VideoTimestampRepo {
	return &videoTimestampRepoImpl{db: db}
}

func (v *videoTimestampRepoImpl) Select(videoId string) ([]*model.VideoTimestampEntity, error) {
	rows, err := v.db.Query(
		context.Background(),
		`SELECT * FROM video_timestamps WHERE video_id = $1`,
		videoId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videoTimestamps := []*model.VideoTimestampEntity{}
	for rows.Next() {
		var videoTImestampEntity model.VideoTimestampEntity
		if err := rows.Scan(
			&videoTImestampEntity.TimestampId,
			&videoTImestampEntity.VideoId,
			&videoTImestampEntity.Timestamp,
			&videoTImestampEntity.Description,
		); err != nil {
			log.Warn(err)
		}
		videoTimestamps = append(videoTimestamps, &videoTImestampEntity)
	}

	return videoTimestamps, nil
}

func (v *videoTimestampRepoImpl) Insert(videoId, timestamp, description string) (*model.VideoTimestampEntity, error) {
	var videoTimestampEntity model.VideoTimestampEntity
	if err := v.db.QueryRow(
		context.Background(),
		`INSERT INTO video_timestamps (
			timestamp_id, video_id, timestamp, description
		) VALUES (
			$1, $2, $3, $4
		)
		RETURNING *`,
		uuid.New().String(), videoId, timestamp, description,
	).Scan(
		&videoTimestampEntity.TimestampId,
		&videoTimestampEntity.VideoId,
		&videoTimestampEntity.Timestamp,
		&videoTimestampEntity.Description,
	); err != nil {
		return nil, err
	}

	return &videoTimestampEntity, nil
}

func (v *videoTimestampRepoImpl) Delete(timestampId string) error {
	_, err := v.db.Exec(context.Background(), `DELETE FROM video_timestamps WHERE timestamp_id = $1`, timestampId)
	return err
}
