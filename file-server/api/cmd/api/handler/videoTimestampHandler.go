package handler

import (
	"pechka/file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func VideoTimestampHandler(app *fiber.App, videoTimestampService *service.VideoTimestampService) {
	app.Get("/api/videos/:videoId/timestamps", func(c *fiber.Ctx) error {
		videoId := c.Params("videoId")
		timestamps, err := videoTimestampService.Select(videoId)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(timestamps)
	})
	app.Post("/api/videos/:videoId/timestamps", func(c *fiber.Ctx) error {
		videoId := c.Params("videoId")
		var body service.VideoTimestampModelForInsert
		if err := c.BodyParser(&body); err != nil {
			return c.SendStatus(400)
		}

		timestamp, err := videoTimestampService.Insert(videoId, &body)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(timestamp)
	})
	app.Delete("/api/videos/:videoId/timestamps/:id", func(c *fiber.Ctx) error {
		timestampId := c.Params("id")
		if err := videoTimestampService.Delete(timestampId); err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(map[string]interface{}{})
	})
}
