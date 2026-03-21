package usecase

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
	"pechka/streaming-service/api/internal/infrastructure/auth"
)

// ---- Request / Response types ----

type SessionRequest struct {
	Email string `json:"email"`
}

type SessionResponse struct {
	AccessToken string `json:"access_token"`
}

type MeResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Groups      []string  `json:"groups"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}

// ---- Interface ----

type AuthUseCase interface {
	Session(ctx context.Context, tokenStr string) (*SessionResponse, error)
	Me(ctx context.Context, accessToken string) (*MeResponse, error)
}

// ---- Implementation ----

type authUseCase struct {
	userRepo  domain.UserRepository
	accessTTL time.Duration
}

func NewAuthUseCase(userRepo domain.UserRepository) AuthUseCase {
	accessMinutes := 60
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			accessMinutes = n
		}
	}

	return &authUseCase{
		userRepo:  userRepo,
		accessTTL: time.Duration(accessMinutes) * time.Minute,
	}
}

func (u *authUseCase) Session(ctx context.Context, tokenStr string) (*SessionResponse, error) {
	if tokenStr == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Unified Verification Logic
	cfClaims, err := auth.VerifyCloudflareJWT(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cloudflare jwt: %w", err)
	}

	email := cfClaims.Email
	if email == "" {
		return nil, fmt.Errorf("email not found in cloudflare jwt")
	}

	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// JIT Provisioning
			user = &domain.User{
				ID:          uuid.Must(uuid.NewV7()),
				Email:       email,
				DisplayName: strings.Split(email, "@")[0],
				Status:      "active",
			}
			if err := u.userRepo.Create(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to JIT create user: %w", err)
			}
			// (Optional) add user to default "Users" group here if you want
		} else {
			return nil, fmt.Errorf("db error finding user: %w", err)
		}
	} else {
		// Update LastLogin
		now := time.Now()
		user.LastLogin = &now
		_ = u.userRepo.Update(ctx, user) // ignore error on best-effort update
	}

	// Fetch RBAC details
	groups, err := u.userRepo.GetGroupsByUserID(ctx, user.ID)
	if err != nil { return nil, fmt.Errorf("failed fetching groups: %w", err) }
	
	roles, err := u.userRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil { return nil, fmt.Errorf("failed fetching roles: %w", err) }
	
	perms, err := u.userRepo.GetPermissionsByUserID(ctx, user.ID)
	if err != nil { return nil, fmt.Errorf("failed fetching permissions: %w", err) }

	groupNames := make([]string, 0)
	for _, g := range groups { groupNames = append(groupNames, g.Name) }

	roleNames := make([]string, 0)
	for _, r := range roles { roleNames = append(roleNames, r.Name) }

	permStrings := make([]string, 0)
	for _, p := range perms { permStrings = append(permStrings, fmt.Sprintf("%s:%s", p.Resource, p.Action)) }

	// Mock Admin Backdoor for local testing
	if strings.HasPrefix(user.Email, "admin-") || user.Email == "admin@example.com" {
		roleNames = append(roleNames, "admin")
	}

	// Issue App JWT
	token, err := auth.GenerateAppJWT(user.ID.String(), user.Email, groupNames, roleNames, permStrings, u.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate app jwt: %w", err)
	}

	return &SessionResponse{AccessToken: token}, nil
}

func (u *authUseCase) Me(ctx context.Context, accessToken string) (*MeResponse, error) {
	claims, err := auth.VerifyAppJWT(accessToken)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("malformed user_id in token")
	}

	return &MeResponse{
		ID:          id,
		Email:       claims.Email,
		DisplayName: strings.Split(claims.Email, "@")[0],
		Groups:      claims.Groups,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}
