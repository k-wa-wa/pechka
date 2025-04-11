package service

import (
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"
	"pechka/file-server/internal/utils"
)

type VideoTimestampService struct {
	VideoTimestampRepo repository.VideoTimestampRepo
}

func (v *VideoTimestampService) Select(videoId string) ([]*model.VideoTimestampEntity, error) {
	return v.VideoTimestampRepo.Select(videoId)
}

type VideoTimestampModelForInsert struct {
	Timestamp   string `json:"timestamp" validate:"required,len=8"`
	Description string `json:"description"`
}

func (v *VideoTimestampService) Insert(videoId string, target *VideoTimestampModelForInsert) (*model.VideoTimestampEntity, error) {
	if err := utils.Validate.Struct(target); err != nil {
		return nil, err
	}

	return v.VideoTimestampRepo.Insert(videoId, target.Timestamp, target.Description)
}

func (v *VideoTimestampService) Delete(timestampId string) error {
	return v.VideoTimestampRepo.Delete(timestampId)
}
