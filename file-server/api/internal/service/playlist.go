package service

import (
	"pechka/file-server/internal/domain/model"
	"pechka/file-server/internal/domain/repository"
)

type PlaylistService struct {
	PlaylistRepo repository.PlaylistRepo
}

type PlaylistModel struct {
	Playlist  *model.PlaylistEntity `json:"playlist"`
	Videos    []*model.VideoEntity  `json:"videos"`
	NumVideos int                   `json:"num_videos"`
}

/* TODO: latest以外も追加する */
func (ps *PlaylistService) Select() ([]*PlaylistModel, error) {
	latestPlaylist, err := ps.SelectOne("latest", 10, 0)
	if err != nil {
		return nil, err
	}

	return []*PlaylistModel{
		latestPlaylist,
	}, nil
}

/* playlistは未実装 */
func (ps *PlaylistService) SelectOne(playlistId string, limit, offset int) (*PlaylistModel, error) {
	videos, err := ps.PlaylistRepo.SelectVideos(playlistId, limit, offset)
	if err != nil {
		return nil, err
	}

	playlist, err := ps.PlaylistRepo.Select(playlistId)
	if err != nil {
		return nil, err
	}

	numVideos, err := ps.PlaylistRepo.CountVideos(playlistId)
	if err != nil {
		return nil, err
	}

	return &PlaylistModel{
		Playlist:  playlist,
		Videos:    videos,
		NumVideos: numVideos,
	}, nil

}
