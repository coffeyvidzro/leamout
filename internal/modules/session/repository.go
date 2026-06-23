package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("session not found")

type CreateParams struct {
	UserID    uuid.UUID
	RawToken  string
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
}

type Repository struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

func NewRepository(db *pgxpool.Pool, cache *redis.Client) *Repository {
	return &Repository{db: db, cache: cache}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (*Session, error) {
	const query = `
INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_seen_at)
VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, NOW())
RETURNING id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at`

	session, err := scanSession(r.db.QueryRow(ctx, query, params.UserID, params.TokenHash, params.UserAgent, params.IPAddress, params.ExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if params.TokenHash != "" && r.cache != nil {
		if err := r.cache.Set(ctx, cacheKey(params.TokenHash), params.UserID.String(), time.Until(params.ExpiresAt)).Err(); err != nil {
			return nil, fmt.Errorf("cache session: %w", err)
		}
	}

	return session, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
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

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE id = $1`

	session, err := scanSession(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}

	return session, nil
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL
RETURNING token_hash`

	var tokenHash string
	if err := r.db.QueryRow(ctx, query, userID, id).Scan(&tokenHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("delete session: %w", err)
	}

	if r.cache != nil {
		if err := r.cache.Del(ctx, cacheKey(tokenHash)).Err(); err != nil {
			return fmt.Errorf("delete cached session: %w", err)
		}
	}

	return nil
}

func (r *Repository) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL
RETURNING token_hash`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete all sessions by user id: %w", err)
	}
	defer rows.Close()

	cacheKeys := make([]string, 0)
	for rows.Next() {
		var tokenHash string
		if err := rows.Scan(&tokenHash); err != nil {
			return fmt.Errorf("scan deleted session token hash: %w", err)
		}
		cacheKeys = append(cacheKeys, cacheKey(tokenHash))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate deleted session token hashes: %w", err)
	}

	if len(cacheKeys) > 0 && r.cache != nil {
		if err := r.cache.Del(ctx, cacheKeys...).Err(); err != nil {
			return fmt.Errorf("delete cached sessions: %w", err)
		}
	}

	return nil
}

func (r *Repository) DeleteByToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}

	tokenHash := HashToken(rawToken)

	if r.cache != nil {
		if err := r.cache.Del(ctx, cacheKey(tokenHash)).Err(); err != nil {
			return fmt.Errorf("delete cached session: %w", err)
		}
	}

	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := r.db.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}

	return nil
}

func cacheKey(tokenHash string) string {
	return "session:" + tokenHash
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
