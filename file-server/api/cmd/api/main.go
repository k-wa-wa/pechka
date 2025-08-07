package main

import (
	"pechka/file-server/cmd/api/handler"
	"pechka/file-server/internal/config"
	"pechka/file-server/internal/db"

	"pechka/file-server/internal/infrastructure"
	"pechka/file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.NewConfig()

	db, err := db.InitDB()
	if err != nil {
		panic(err)
	}

	app := fiber.New()
	app.Use(logger.New(logger.Config{
		Format:     "${ip} - - [${time}] \"${method} ${path} ${protocol}\" ${status} ${bytesSent} \"${referer}\" \"${ua}\" \"${reqHeader:x-request-id}\" ${latency}\n",
		TimeFormat: "02/Jan/2006:15:04:05 -0700",
	}))

	app.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	app.Static("/resources/hls", cfg.HlsResourceDir)

	playlistRepo := infrastructure.NewPlaylistRepo(db)
	videoRepo := infrastructure.NewVideoRepo(db)
	videoTimestampRepo := infrastructure.NewVideoTimestampRepo(db)

	playlistService := &service.PlaylistService{PlaylistRepo: playlistRepo}
	videoService := &service.VideoService{VideoRepo: videoRepo}
	videoTimestampService := &service.VideoTimestampService{VideoTimestampRepo: videoTimestampRepo}

	handler.PlaylistHandler(app, playlistService)
	handler.VideoHandler(app, videoService)
	handler.VideoTimestampHandler(app, videoTimestampService)

	log.Fatal(app.Listen(":8000"))
}
