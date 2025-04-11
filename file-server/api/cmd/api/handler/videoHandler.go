package handler

import (
	"pechka/file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func VideoHandler(app *fiber.App, videoService *service.VideoService) {
	app.Get("/api/videos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		video, err := videoService.SelectOne(id)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(video)
	})
	app.Put("/api/videos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var body service.VideoModelForUpdate
		if err := c.BodyParser(&body); err != nil {
			return c.SendStatus(400)
		}

		video, err := videoService.Update(id, &body)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}

		return c.JSON(video)
	})
}
