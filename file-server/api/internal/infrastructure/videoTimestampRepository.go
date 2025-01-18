package infrastructure

import (
	"context"
	"pechka/file-server/internal/db"

	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

type VideoTimestampEntity struct {
	TimestampId string `json:"timestampId"`
	VideoId     string `json:"videoId"`
	Timestamp   string `json:"timestamp"`
	Description string `json:"description"`
}

type VideoTimestampRepo interface {
	Select(videoId string) ([]*VideoTimestampEntity, error)
	Insert(videoId, timestamp, description string) (*VideoTimestampEntity, error)
	Delete(timestampId string) error
}

type VideoTimestampRepoImpl struct {
	Db db.DB
}

func (v *VideoTimestampRepoImpl) Select(videoId string) ([]*VideoTimestampEntity, error) {
	rows, err := v.Db.Query(
		context.Background(),
		`SELECT * FROM video_timestamps WHERE video_id = $1`,
		videoId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videoTimestamps := []*VideoTimestampEntity{}
	for rows.Next() {
		var videoTImestampEntity VideoTimestampEntity
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

func (v *VideoTimestampRepoImpl) Insert(videoId, timestamp, description string) (*VideoTimestampEntity, error) {
	var videoTimestampEntity VideoTimestampEntity
	if err := v.Db.QueryRow(
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

func (v *VideoTimestampRepoImpl) Delete(timestampId string) error {
	_, err := v.Db.Exec(context.Background(), `DELETE FROM video_timestamps WHERE timestamp_id = $1`, timestampId)
	return err
}
