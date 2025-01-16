package service

import (
	"pechka/file-server/internal/infrastructure"

	"github.com/gofiber/fiber/v2/log"
)

type PlaylistService struct {
	VideoRepo infrastructure.VideoRepo
}

type PlaylistModel struct {
	Title  string        `json:"title"`
	Videos []*VideoModel `json:"videos"`
}

func (ps *PlaylistService) Get() ([]*PlaylistModel, error) {
	playlists := []*PlaylistModel{}

	// latest
	latestVideos, err := ps.VideoRepo.SelectLatest()
	if err != nil {
		log.Warn(err)
	} else {
		playlists = append(
			playlists,
			&PlaylistModel{
				Title:  "Latest",
				Videos: videoEntitiesToVideoModel(latestVideos),
			},
		)
	}

	return playlists, nil
}
