package customLogger

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func headersToString(_headers map[string][]string) string {
	headers := make([]string, 0)
	for k, v := range _headers {
		headers = append(headers, k+"="+strings.Join(v, ","))
	}
	return strings.Join(headers, "&")
}

func NewLogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqId := c.Locals("requestid").(string)

		slog.Info(
			"req",
			slog.Group("_data", slog.String("reqId", reqId),
				slog.String("reqHeader", headersToString(c.GetReqHeaders())),
				slog.String("queryParams", c.Request().URI().QueryArgs().String()),
				slog.String("reqBody", string(c.Body())),
			),
		)

		c.Next()
		slog.Info("res", slog.String("reqId", reqId))
		return nil
	}
}
