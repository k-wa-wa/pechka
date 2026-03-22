package auth_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"pechka/streaming-service/api/internal/usecase"
)

// GET /api/v1/auth/session

func TestSession_Returns401WhenCfHeaderIsMissing(t *testing.T) {
	resp, err := newApp(&mockAuthUseCase{}).Test(httptest.NewRequest("GET", "/api/v1/auth/session", nil))
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 401, resp)
}

func TestSession_Returns500WhenUsecaseFails(t *testing.T) {
	mock := &mockAuthUseCase{
		sessionFn: func(_ context.Context, _ string) (*usecase.SessionResponse, error) {
			return nil, fmt.Errorf("invalid cloudflare jwt")
		},
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.Header.Set("Cf-Access-Jwt-Assertion", "dummy.cf.token")

	resp, err := newApp(mock).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 500, resp)
}

func TestSession_Returns200WithAccessToken(t *testing.T) {
	mock := &mockAuthUseCase{
		sessionFn: func(_ context.Context, _ string) (*usecase.SessionResponse, error) {
			return &usecase.SessionResponse{AccessToken: "app.jwt.token"}, nil
		},
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.Header.Set("Cf-Access-Jwt-Assertion", "valid.cf.token")

	resp, err := newApp(mock).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 200, resp)
}
