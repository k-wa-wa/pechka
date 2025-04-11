package repository

import "pechka/file-server/internal/domain/model"

type PlaylistRepo interface {
	Select(playlistId string) (*model.PlaylistEntity, error)
	SelectVideos(playlistId string, limit, offset int) ([]*model.VideoEntity, error)
	CountVideos(playlistId string) (int, error)
}
