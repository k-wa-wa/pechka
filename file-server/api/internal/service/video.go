package service

import (
	"pechka/file-server/internal/infrastructure"
)

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

func (vs *VideoService) GetOne(id string) (*VideoModel, error) {
	videoEntity, err := vs.VideoRepo.SelectOne(id)
	if err != nil {
		return nil, err
	}

	return &VideoModel{VideoEntity: videoEntity}, nil
}

type GetRes struct {
	Videos []*VideoModel `json:"videos"`
	NextId string        `json:"nextId"`
}

func (vs *VideoService) Get(playlistId, fromId string) (*GetRes, error) {
	limit := 10

	videoEntities, err := vs.VideoRepo.Select(playlistId, fromId, limit+1)
	if err != nil {
		return nil, err
	}

	res := &GetRes{}
	if len(videoEntities) > limit {
		res.NextId = videoEntities[limit].Id
		videoEntities = videoEntities[:limit]
	}
	res.Videos = videoEntitiesToVideoModel(videoEntities)

	return res, nil
}

type VideoModelForPut struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (vs *VideoService) Put(id string, target *VideoModelForPut) (*VideoModel, error) {
	videoEntity, err := vs.VideoRepo.Update(id, target.Title, target.Description)
	if err != nil {
		return nil, err
	}

	return &VideoModel{VideoEntity: videoEntity}, nil
}
