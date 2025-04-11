package repository

import "pechka/file-server/internal/domain/model"

type VideoTimestampRepo interface {
	Select(videoId string) ([]*model.VideoTimestampEntity, error)
	Insert(videoId, timestamp, description string) (*model.VideoTimestampEntity, error)
	Delete(timestampId string) error
}
