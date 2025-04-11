package service

import (
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"
)

type VideoService struct {
	VideoRepo repository.VideoRepo
}

func (vs *VideoService) SelectOne(id string) (*model.VideoEntity, error) {
	return vs.VideoRepo.SelectOne(id)
}

type VideoModelForUpdate struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (vs *VideoService) Update(id string, target *VideoModelForUpdate) (*model.VideoEntity, error) {
	return vs.VideoRepo.Update(id, target.Title, target.Description)
}
