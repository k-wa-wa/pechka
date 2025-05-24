package main

import (
	"context"
	"pechka/file-server/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	db, err := db.InitDB()
	if err != nil {
		panic(err)
	}

	if err := refreshLatestPlaylist(context.Background(), db); err != nil {
		panic(err)
	}
}

func refreshLatestPlaylist(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	latestPlaylistId := "latest"
	if _, err := tx.Exec(ctx, `
		DELETE FROM playlists WHERE id = $1;
	`, latestPlaylistId); err != nil {
		tx.Rollback(ctx)
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO playlists (id, title, description)
		VALUES ($1, $2, $3)
	`, latestPlaylistId, "Latest", ""); err != nil {
		tx.Rollback(ctx)
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO playlist_videos (playlist_id, video_id)
		SELECT $1, id
		FROM videos
		ORDER BY updated_at DESC
	`, latestPlaylistId); err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
