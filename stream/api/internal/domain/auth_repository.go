package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines persistence operations for users and RBAC
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	
	// RBAC related
	GetPermissionsByUserID(ctx context.Context, userID uuid.UUID) ([]Permission, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]Role, error)
	GetGroupsByUserID(ctx context.Context, userID uuid.UUID) ([]Group, error)
}
