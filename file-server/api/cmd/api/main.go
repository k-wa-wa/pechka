package main

import (
	"log/slog"
	"os"
	"pechka/file-server/internal/config"
	"pechka/file-server/internal/db"
	"pechka/file-server/internal/infrastructure"
	"pechka/file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
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
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(playlists)
	})

	videoService := service.VideoService{VideoRepo: videoRepo}
	app.Get("/api/videos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		video, err := videoService.Get(id)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(video)
	})
	app.Put("/api/videos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var body service.VideoModelForPut
		if err := c.BodyParser(&body); err != nil {
			return c.SendStatus(400)
		}

		video, err := videoService.Put(id, &body)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}

		return c.JSON(video)
	})

	videoTimestampRepo := &infrastructure.VideoTimestampRepoImpl{Db: db}
	videoTimestampService := service.VideoTimestampService{VideoTimestampRepo: videoTimestampRepo}
	app.Get("/api/video-timestamps/:id", func(c *fiber.Ctx) error {
		videoId := c.Params("id")
		timestamps, err := videoTimestampService.Get(videoId)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(timestamps)
	})
	app.Post("/api/video-timestamps/:id", func(c *fiber.Ctx) error {
		videoId := c.Params("id")
		var body service.VideoTimestampModelForPost
		if err := c.BodyParser(&body); err != nil {
			return c.SendStatus(400)
		}

		timestamp, err := videoTimestampService.Post(videoId, &body)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(timestamp)
	})
	app.Delete("/api/video-timestamps/:id", func(c *fiber.Ctx) error {
		timestampId := c.Params("id")
		if err := videoTimestampService.Delete(timestampId); err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(map[string]interface{}{})
	})

	log.Fatal(app.Listen(":8000"))
}
