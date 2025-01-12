package main

import (
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"pechka/file-server/internal/config"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type HlsResource struct {
	Path string `json:"path"`
}

func getHlsResources(hlsResourceDir string) ([]*HlsResource, error) {
	hlsResources := []*HlsResource{}

	pattern := "*.m3u8"
	matches := []string{}

	if err := filepath.Walk(hlsResourceDir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, fileInfo.Name())
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, path)
		}

		return nil
	}); err != nil {
		return hlsResources, err
	}

	for _, match := range matches {
		hlsResources = append(
			hlsResources,
			&HlsResource{
				Path: filepath.Join("/resources/hls", strings.ReplaceAll(match, hlsResourceDir, "")),
			},
		)
	}

	return hlsResources, nil
}

func main() {
	cfg := config.NewConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	app := fiber.New()

	app.Use(requestid.New())
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	app.Static("/resources/hls", cfg.HlsResourceDir)
	app.Get("/api/hls/hls-resources", func(c *fiber.Ctx) error {
		hlsResources, err := getHlsResources(cfg.HlsResourceDir)
		if err != nil {
			log.Fatal(err)
		}

		return c.JSON(hlsResources)
	})

	log.Fatal(app.Listen(":8000"))
}
