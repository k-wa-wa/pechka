package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pechka/streaming-service/api/internal/domain"
)

// ---- Request / Response types ----

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type MeResponse struct {
	ID    uuid.UUID       `json:"id"`
	Email string          `json:"email"`
	Role  domain.UserRole `json:"role"`
}

// ---- Interface ----

type AuthUseCase interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthTokens, error)
	Login(ctx context.Context, req LoginRequest) (*AuthTokens, error)
	Refresh(ctx context.Context, req RefreshRequest) (*AuthTokens, error)
	// Me validates the access token and returns the claims (no DB hit)
	Me(ctx context.Context, accessToken string) (*MeResponse, error)
}

// ---- Implementation ----

type authUseCase struct {
	userRepo  domain.UserRepository
	tokenRepo domain.TokenRepository
	jwtSecret []byte
	accessTTL time.Duration
	refreshTTL time.Duration
}

type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthUseCase(userRepo domain.UserRepository, tokenRepo domain.TokenRepository) AuthUseCase {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change_me_in_production"
	}

	accessMinutes := 15
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			accessMinutes = n
		}
	}

	refreshDays := 30
	if v := os.Getenv("JWT_REFRESH_TTL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			refreshDays = n
		}
	}

	return &authUseCase{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  []byte(secret),
		accessTTL:  time.Duration(accessMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

func (u *authUseCase) Register(ctx context.Context, req RegisterRequest) (*AuthTokens, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         domain.UserRoleUser,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return u.issueTokens(ctx, user)
}

func (u *authUseCase) Login(ctx context.Context, req LoginRequest) (*AuthTokens, error) {
	user, err := u.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// mask not-found as invalid credentials
		return nil, fmt.Errorf("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	return u.issueTokens(ctx, user)
}

func (u *authUseCase) Refresh(ctx context.Context, req RefreshRequest) (*AuthTokens, error) {
	tokenHash := hashToken(req.RefreshToken)

	stored, err := u.tokenRepo.FindAndDelete(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	user, err := u.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return u.issueTokens(ctx, user)
}

func (u *authUseCase) Me(ctx context.Context, accessToken string) (*MeResponse, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return u.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}

	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("malformed token claims")
	}

	return &MeResponse{
		ID:    id,
		Email: claims.Email,
		Role:  domain.UserRole(claims.Role),
	}, nil
}

// issueTokens generates a new JWT access token and a refresh token, then persists the refresh token.
func (u *authUseCase) issueTokens(ctx context.Context, user *domain.User) (*AuthTokens, error) {
	// 1. Access Token (JWT)
	now := time.Now()
	claims := jwtClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(u.accessTTL)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(u.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Refresh Token (opaque random bytes, stored as hash)
	rawRefresh, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshRecord := &domain.RefreshToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: now.Add(u.refreshTTL),
	}

	if err := u.tokenRepo.Save(ctx, refreshRecord); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// generateOpaqueToken creates a cryptographically random 32-byte hex string.
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken SHA-256 hashes a raw token string for safe DB storage.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
