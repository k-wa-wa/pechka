package repository

import "pechka/file-server/internal/domain/model"

type VideoRepo interface {
	SelectOne(id string) (*model.VideoEntity, error)
	Update(id, title, description string) (*model.VideoEntity, error)
}
