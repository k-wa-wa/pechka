package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pechka/streaming-service/api/internal/domain"
)

// userRepository implements domain.UserRepository
type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) domain.UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, display_name, avatar_url, status, last_login, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.LastLogin,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users 
		SET email = $2, display_name = $3, avatar_url = $4, status = $5, last_login = $6, updated_at = $7
		WHERE id = $1
	`
	user.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.LastLogin,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, status, last_login, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.Status, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, status, last_login, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.Status, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return &u, nil
}

func (r *userRepository) GetPermissionsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		LEFT JOIN user_roles ur ON rp.role_id = ur.role_id AND ur.user_id = $1
		LEFT JOIN group_roles gr ON rp.role_id = gr.role_id
		LEFT JOIN user_groups ug ON gr.group_id = ug.group_id AND ug.user_id = $1
		WHERE ur.user_id IS NOT NULL OR ug.user_id IS NOT NULL
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *userRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	query := `
		SELECT DISTINCT r.id, r.name, r.description, r.created_at
		FROM roles r
		LEFT JOIN user_roles ur ON r.id = ur.role_id AND ur.user_id = $1
		LEFT JOIN group_roles gr ON r.id = gr.role_id
		LEFT JOIN user_groups ug ON gr.group_id = ug.group_id AND ug.user_id = $1
		WHERE ur.user_id IS NOT NULL OR ug.user_id IS NOT NULL
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var r domain.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func (r *userRepository) GetGroupsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Group, error) {
	query := `
		SELECT g.id, g.name, g.description, g.created_at
		FROM groups g
		JOIN user_groups ug ON g.id = ug.group_id
		WHERE ug.user_id = $1
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}
