package main

import (
	"log"
	"log/slog"
	"os"
	"pechka/file-server/internal/config"
	"pechka/file-server/internal/db"
	"pechka/file-server/internal/infrastructure"
	"pechka/file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	cfg := config.NewConfig()

	db, err := db.InitDB()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	app := fiber.New()

	app.Use(requestid.New())
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	app.Static("/resources/hls", cfg.HlsResourceDir)

	videoRepo := &infrastructure.VideoRepoImpl{Db: db}
	playlistService := service.PlaylistService{VideoRepo: videoRepo}
	app.Get("/api/playlists", func(c *fiber.Ctx) error {
		playlists, err := playlistService.Get()
		if err != nil {
			log.Println(err)
			return c.SendStatus(400)
		}
		return c.JSON(playlists)
	})

	videoService := service.VideoService{VideoRepo: videoRepo}
	app.Get("/api/videos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		video, err := videoService.Get(id)
		if err != nil {
			log.Println(err)
			return c.SendStatus(400)
		}
		return c.JSON(video)
	})

	log.Fatal(app.Listen(":8000"))
}
