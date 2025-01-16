package service

import "pechka/file-server/internal/infrastructure"

type VideoService struct {
	VideoRepo infrastructure.VideoRepo
}

type VideoModel struct {
	*infrastructure.VideoEntity
}

func videoEntitiesToVideoModel(videoEntities []*infrastructure.VideoEntity) []*VideoModel {
	videoModels := []*VideoModel{}
	for _, videoEntity := range videoEntities {
		videoModels = append(videoModels, &VideoModel{VideoEntity: videoEntity})
	}
	return videoModels
}

func (vs *VideoService) Get(id string) (*VideoModel, error) {
	videoEntity, err := vs.VideoRepo.Select(id)
	if err != nil {
		return nil, err
	}

	return &VideoModel{VideoEntity: videoEntity}, nil
}
