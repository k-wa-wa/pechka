package auth

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type AppClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Groups      []string `json:"groups"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

type CloudflareClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

var (
	jwtSecret []byte
	cfKeyfunc keyfunc.Keyfunc
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change_me_in_production"
	}
	jwtSecret = []byte(secret)

	jwksURL := os.Getenv("CLOUDFLARE_JWKS_URL")
	if jwksURL != "" {
		var err error
		cfKeyfunc, err = keyfunc.NewDefault([]string{jwksURL})
		if err != nil {
			log.Printf("failed to create Cloudflare keyfunc: %v", err)
		}
	}
}

func GenerateAppJWT(userID, email string, groups, roles, permissions []string, ttl time.Duration) (string, error) {
	now := time.Now()
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "pechka-auth-service"
	}

	claims := AppClaims{
		UserID:      userID,
		Email:       email,
		Groups:      groups,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyAppJWT(tokenString string) (*AppClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AppClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AppClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func VerifyCloudflareJWT(tokenString string) (*CloudflareClaims, error) {
	expectedIssuer := os.Getenv("CLOUDFLARE_JWT_ISSUER")
	expectedAudience := os.Getenv("CLOUDFLARE_JWT_AUDIENCE")

	if cfKeyfunc == nil {
		// Fallback for local development if JWKS is not pre-initialized but URL is provided
		jwksURL := os.Getenv("CLOUDFLARE_JWKS_URL")
		if jwksURL != "" {
			var err error
			cfKeyfunc, err = keyfunc.NewDefault([]string{jwksURL})
			if err != nil {
				return nil, fmt.Errorf("failed to initialize keyfunc: %w", err)
			}
		} else {
			return nil, fmt.Errorf("CLOUDFLARE_JWKS_URL is not set")
		}
	}

	token, err := jwt.ParseWithClaims(tokenString, &CloudflareClaims{}, cfKeyfunc.Keyfunc)
	if err != nil {
		log.Printf("cloudflare jwt parse error: %v", err)
		return nil, fmt.Errorf("failed to parse cloudflare jwt: %w", err)
	}

	claims, ok := token.Claims.(*CloudflareClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid cloudflare jwt")
	}

	if expectedIssuer != "" && claims.Issuer != expectedIssuer {
		log.Printf("cloudflare jwt issuer mismatch: got %v, want %v", claims.Issuer, expectedIssuer)
		return nil, fmt.Errorf("invalid issuer: got %s, want %s", claims.Issuer, expectedIssuer)
	}

	audMatch := false
	if expectedAudience == "" {
		audMatch = true
	} else {
		for _, aud := range claims.Audience {
			if aud == expectedAudience {
				audMatch = true
				break
			}
		}
	}

	if !audMatch {
		log.Printf("cloudflare jwt audience mismatch: got %v, want %s", claims.Audience, expectedAudience)
		return nil, fmt.Errorf("invalid audience: want %s", expectedAudience)
	}

	return claims, nil
}
