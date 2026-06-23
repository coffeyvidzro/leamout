package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("session not found")

type Repository interface {
	Create(ctx context.Context, params CreateParams) (*Session, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Session, error)
	RevokeByID(ctx context.Context, userID, id uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
}

type CreateParams struct {
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, params CreateParams) (*Session, error) {
	const query = `
INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_seen_at)
VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, NOW())
RETURNING id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at`

	session, err := scanSession(r.pool.QueryRow(ctx, query, params.UserID, params.TokenHash, params.UserAgent, params.IPAddress, params.ExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

func (r *PostgresRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions by user id: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, *session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE id = $1`

	session, err := scanSession(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}

	return session, nil
}

func (r *PostgresRepository) RevokeByID(ctx context.Context, userID, id uuid.UUID) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL`

	result, err := r.pool.Exec(ctx, query, userID, id)
	if err != nil {
		return fmt.Errorf("revoke session by id: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL`

	if _, err := r.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("revoke all sessions by user id: %w", err)
	}

	return nil
}

func (r *PostgresRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := r.pool.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoke session by token hash: %w", err)
	}

	return nil
}

func scanSession(row pgx.Row) (*Session, error) {
	var session Session
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.LastSeenAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &session, nil
}
