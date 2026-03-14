package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines persistence operations for users
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// TokenRepository defines persistence operations for refresh tokens
type TokenRepository interface {
	Save(ctx context.Context, token *RefreshToken) error
	// FindAndDelete performs an atomic lookup + delete (token rotation).
	// Returns the stored RefreshToken record if the hash matches, or an error.
	FindAndDelete(ctx context.Context, tokenHash string) (*RefreshToken, error)
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}
