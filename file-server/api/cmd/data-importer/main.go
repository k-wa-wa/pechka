package main

import (
	"context"
	"log"
	"path/filepath"
	"pechka/file-server/internal/config"
	"pechka/file-server/internal/db"
	"pechka/file-server/pkg/fileutil"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.NewConfig()

	db, err := db.InitDB()
	if err != nil {
		panic(err)
	}

	if err := importHlsData(cfg.HlsResourceDir, db); err != nil {
		panic(err)
	}
}

func importHlsData(hlsResourceDir string, db *pgxpool.Pool) error {
	hlsResourceDirFullPath, err := filepath.Abs(hlsResourceDir)
	if err != nil {
		return err
	}

	hlsFiles, err := fileutil.GlobFiles(hlsResourceDirFullPath, "*.m3u8")
	if err != nil {
		return err
	}

	for _, hlsFile := range hlsFiles {
		_, err := db.Exec(
			context.Background(),
			`
			INSERT INTO videos (
				id, fullpath, title, description, url
			) VALUES (
			 	$1, $2, $3, $4, $5
			) ON CONFLICT (fullpath) DO NOTHING
			`,
			uuid.New().String(),
			hlsFile,
			filepath.Base(hlsFile),
			"",
			filepath.Join("/resources/hls", strings.ReplaceAll(hlsFile, hlsResourceDirFullPath, "")),
		)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
