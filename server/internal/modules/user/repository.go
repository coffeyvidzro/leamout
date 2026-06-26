package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = `
SELECT id, name, email, email_verified, avatar_url, status, created_at, updated_at
FROM users
WHERE id = $1 AND status <> 'deleted'`

	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) error {
	updates := make([]string, 0, 2)
	args := []any{id}

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.AvatarURL != nil {
		updates = append(updates, fmt.Sprintf("avatar_url = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.AvatarURL))
	}
	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $1 AND status <> 'deleted'",
		strings.Join(updates, ", "),
	)
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE users
SET status = 'deleted'
WHERE id = $1 AND status <> 'deleted'`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.EmailVerified,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}
