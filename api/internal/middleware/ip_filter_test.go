package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestIPFilter(t *testing.T) {
	tests := []struct {
		name          string
		allowedRanges string
		clientIP      string
		expectedCode  int
	}{
		{
			name:          "empty allowed ranges denies all",
			allowedRanges: "",
			clientIP:      "127.0.0.1",
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "single allowed CIDR matches",
			allowedRanges: "192.168.1.0/24",
			clientIP:      "192.168.1.50",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "single allowed CIDR rejects outside IP",
			allowedRanges: "192.168.1.0/24",
			clientIP:      "10.0.0.1",
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "multiple CIDRs with whitespace and invalid entry",
			allowedRanges: "192.168.1.0/24, invalid-cidr, 10.0.0.0/8 , ",
			clientIP:      "10.1.2.3",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "invalid client IP header",
			allowedRanges: "10.0.0.0/8",
			clientIP:      "invalid-ip",
			expectedCode:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(echo.HeaderXRealIP, tt.clientIP)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := IPFilter(tt.allowedRanges)(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedCode {
						t.Fatalf("expected HTTP status %d, got %d", tt.expectedCode, he.Code)
					}
					return
				}
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tt.expectedCode {
				t.Fatalf("expected HTTP status %d, got %d", tt.expectedCode, rec.Code)
			}
		})
	}
}
