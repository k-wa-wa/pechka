package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// IPFilter returns a middleware that limits requests to allowed CIDR blocks.
// Multiple ranges can be separated by commas (e.g. "192.168.1.0/24,10.0.0.0/8").
// If allowedRanges is empty, all requests are forbidden for safety.
func IPFilter(allowedRanges string) echo.MiddlewareFunc {
	var ipNets []*net.IPNet

	if allowedRanges != "" {
		parts := strings.Split(allowedRanges, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(trimmed)
			if err != nil {
				slog.Error("Failed to parse CIDR in ALLOWED_IP_RANGE", "cidr", trimmed, "error", err)
				continue
			}
			ipNets = append(ipNets, ipNet)
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIPStr := c.RealIP()
			clientIP := net.ParseIP(clientIPStr)
			if clientIP == nil {
				slog.Warn("Rejected request due to unparseable client IP", "ip", clientIPStr)
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
			}

			// If no allowed range configured, forbid all.
			if len(ipNets) == 0 {
				slog.Warn("Rejected request because ALLOWED_IP_RANGE is not configured or empty", "ip", clientIPStr)
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
			}

			allowed := false
			for _, ipNet := range ipNets {
				if ipNet.Contains(clientIP) {
					allowed = true
					break
				}
			}

			if !allowed {
				slog.Warn("Rejected request from unauthorized IP", "ip", clientIPStr)
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
			}

			return next(c)
		}
	}
}
