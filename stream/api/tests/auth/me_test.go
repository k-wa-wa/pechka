package auth_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	infraAuth "pechka/streaming-service/api/internal/infrastructure/auth"
	"pechka/streaming-service/api/internal/usecase"
)

// GET /api/v1/auth/me

func TestMe_Returns401WhenAuthorizationHeaderIsMissing(t *testing.T) {
	resp, err := newApp(&mockAuthUseCase{}).Test(httptest.NewRequest("GET", "/api/v1/auth/me", nil))
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 401, resp)
}

func TestMe_Returns401WhenAuthorizationHeaderIsMalformed(t *testing.T) {
	cases := []string{
		"",
		"Token abc",
		"bearer abc", // lowercase
		"Bearertoken",
	}
	for _, header := range cases {
		t.Run(fmt.Sprintf("header=%q", header), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			resp, err := newApp(&mockAuthUseCase{}).Test(req)
			if err != nil {
				t.Fatal(err)
			}
			assertStatus(t, 401, resp)
		})
	}
}

func TestMe_Returns401WhenTokenIsInvalid(t *testing.T) {
	mock := &mockAuthUseCase{
		meFn: func(_ context.Context, _ string) (*usecase.MeResponse, error) {
			return nil, fmt.Errorf("unauthorized: invalid token")
		},
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")

	resp, err := newApp(mock).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 401, resp)
}

func TestMe_Returns200WithValidToken(t *testing.T) {
	userID := uuid.New()
	mock := &mockAuthUseCase{
		meFn: func(_ context.Context, _ string) (*usecase.MeResponse, error) {
			return &usecase.MeResponse{
				ID:          userID,
				Email:       "nfs-admin@example.com",
				DisplayName: "nfs-admin",
				Roles:       []string{"admin"},
				Groups:      []string{"nfs-admin"},
				Permissions: []string{"content:read", "content:write"},
			}, nil
		},
	}

	token, err := infraAuth.GenerateAppJWT(
		userID.String(), "nfs-admin@example.com",
		[]string{"nfs-admin"}, []string{"admin"}, []string{"content:read"},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := newApp(mock).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, 200, resp)
}
