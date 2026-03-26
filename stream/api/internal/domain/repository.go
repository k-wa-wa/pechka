package domain

import (
	"context"

	"github.com/google/uuid"
)

// ContentRepository defines write-side (PostgreSQL) operations for Content
type ContentRepository interface {
	// Content CRUD
	CreateContent(ctx context.Context, c *Content) error
	GetContentByShortID(ctx context.Context, shortID string) (*Content, error)
	GetContentByID(ctx context.Context, id uuid.UUID) (*Content, error)
	UpdateContent(ctx context.Context, c *Content) error
	ListContents(ctx context.Context) ([]*Content, error)

	// Asset operations (assets table)
	AddAssets(ctx context.Context, contentID uuid.UUID, assets []Asset) error

	// Group permissions (content_group_permissions table)
	GetGroupPermissions(ctx context.Context, contentID uuid.UUID) ([]GroupPermission, error)
	SetGroupPermissions(ctx context.Context, contentID uuid.UUID, perms []GroupPermission) error

	// Utility
	CheckDuplicateByS3Key(ctx context.Context, s3Key string) (bool, error)
	ListAllShortIDs(ctx context.Context) ([]string, error)
	GetGroupNamesByIDs(ctx context.Context, ids []uuid.UUID) ([]string, error)
}
