package handler

import (
	"pechka/file-server/internal/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func PlaylistHandler(app *fiber.App, playlistService *service.PlaylistService) {
	app.Get("/api/playlists", func(c *fiber.Ctx) error {
		playlists, err := playlistService.Select()
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(playlists)
	})

	app.Get("/api/playlists/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		limit, err := strconv.Atoi(c.Query("limit", "10"))
		if err != nil {
			return c.SendStatus(400)
		}
		offset, err := strconv.Atoi(c.Query("offset", "0"))
		if err != nil {
			return c.SendStatus(400)
		}

		playlist, err := playlistService.SelectOne(id, limit, offset)
		if err != nil {
			log.Warn(err)
			return c.SendStatus(400)
		}
		return c.JSON(playlist)
	})
}
