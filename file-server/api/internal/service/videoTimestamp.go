package service

import (
	"pechka/file-server/internal/infrastructure"
	"pechka/file-server/internal/utils"
)

type VideoTimestampService struct {
	VideoTimestampRepo infrastructure.VideoTimestampRepo
}

type VideoTimestampModel struct {
	*infrastructure.VideoTimestampEntity
}

func (v *VideoTimestampService) Get(videoId string) ([]*VideoTimestampModel, error) {
	videoTimestampEntities, err := v.VideoTimestampRepo.Select(videoId)
	if err != nil {
		return nil, err
	}

	videoTimestampModels := []*VideoTimestampModel{}
	for _, videoTimestampEntity := range videoTimestampEntities {
		videoTimestampModels = append(videoTimestampModels, &VideoTimestampModel{VideoTimestampEntity: videoTimestampEntity})
	}

	return videoTimestampModels, nil
}

type VideoTimestampModelForPost struct {
	Timestamp   string `json:"timestamp" validate:"required,len=8"`
	Description string `json:"description"`
}

func (v *VideoTimestampService) Post(videoId string, target *VideoTimestampModelForPost) (*VideoTimestampModel, error) {
	if err := utils.Validate.Struct(target); err != nil {
		return nil, err
	}

	videoTImestampEntity, err := v.VideoTimestampRepo.Insert(videoId, target.Timestamp, target.Description)
	if err != nil {
		return nil, err
	}

	return &VideoTimestampModel{VideoTimestampEntity: videoTImestampEntity}, nil
}

func (v *VideoTimestampService) Delete(timestampId string) error {
	return v.VideoTimestampRepo.Delete(timestampId)
}
