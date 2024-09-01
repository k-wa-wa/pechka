package main

import (
	"bufio"
	"log/slog"
	"os"
	"pechka/api-notifications/pkg/auth"
	"pechka/api-notifications/pkg/customLogger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

type NotificationData struct {
	Data []byte
}

type NotificationChan struct {
	TargetUserId string
	Ch           chan *NotificationData
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	app := fiber.New()

	app.Use(requestid.New())
	app.Use(customLogger.NewLogMiddleware())
	app.Use(auth.AuthConfig)
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		ReadinessEndpoint: "/ready",
	}))

	notificationChannels := map[string]*NotificationChan{}

	app.Get("/api/auth/notifications", func(c *fiber.Ctx) error {
		ch := make(chan *NotificationData)
		connectionId := uuid.New().String()
		notificationChannels[connectionId] = &NotificationChan{
			TargetUserId: "",
			Ch:           ch,
		}

		c.Response().Header.Set("Content-Type", "text/event-stream")
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for notification := range ch {
				data := append([]byte("data: "), notification.Data...)
				data = append(data, []byte("\n\n")...)

				w.Write(data)
				w.Flush()
			}
		})

		// TODO: コネクションを一定時間で切断する、使われていないnotificationChanを削除する
		return nil
	})

	app.Post("/api/webhook/switch-bot", func(c *fiber.Ctx) error {
		for _, ch := range notificationChannels {
			if ch.TargetUserId == "" {
				ch.Ch <- &NotificationData{
					Data: c.Body(),
				}
			}
		}
		return c.SendStatus(200)
	})

	log.Fatal(app.Listen(":8002"))
}
