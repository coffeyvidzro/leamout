package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type CreateSessionParams struct {
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
}

type Repository struct {
	dbPool *pgxpool.Pool
}

func NewRepository(dbPool *pgxpool.Pool) *Repository {
	return &Repository{dbPool: dbPool}
}

func (r *Repository) UpsertOAuthUser(ctx context.Context, profile *oauth.Profile) (*AuthUser, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Default().Error("error rolling back transaction", slog.Any("error", err))
		}
	}()

	user, err := getUserByAccount(ctx, tx, profile.Provider, profile.ProviderUserID)
	if err == nil {
		if err := upsertAccount(ctx, tx, user.ID, profile); err != nil {
			return nil, err
		}
		return user, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	user, err = getUserByEmail(ctx, tx, profile.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		user, err = createOAuthUser(ctx, tx, profile)
		if err != nil {
			return nil, err
		}
	}

	if err := upsertAccount(ctx, tx, user.ID, profile); err != nil {
		return nil, err
	}

	return user, tx.Commit(ctx)
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = `
SELECT id, name, email, email_verified, avatar_url, password_hash, status, created_at, updated_at
FROM users
WHERE id = $1 AND status <> 'deleted'`

	user, err := scanUser(r.dbPool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *Repository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE token_hash = $1`

	session, err := scanSession(r.dbPool.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session by token hash: %w", err)
	}

	return session, nil
}

func (r *Repository) TouchSession(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1`

	if _, err := r.dbPool.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return nil
}

func getUserByAccount(ctx context.Context, tx pgx.Tx, provider, providerUserID string) (*AuthUser, error) {
	const query = `
SELECT u.id, u.name, u.email, u.email_verified, u.avatar_url, u.status
FROM accounts a
JOIN users u ON u.id = a.user_id
WHERE a.provider = $1 AND a.provider_user_id = $2 AND u.status <> 'deleted'`

	return scanAuthUser(tx.QueryRow(ctx, query, provider, providerUserID))
}

func getUserByEmail(ctx context.Context, tx pgx.Tx, email string) (*AuthUser, error) {
	const query = `
SELECT id, name, email, email_verified, avatar_url, status
FROM users
WHERE email = $1 AND status <> 'deleted'`

	return scanAuthUser(tx.QueryRow(ctx, query, email))
}

func createOAuthUser(ctx context.Context, tx pgx.Tx, profile *oauth.Profile) (*AuthUser, error) {
	const query = `
INSERT INTO users (name, email, email_verified, avatar_url)
VALUES (COALESCE(NULLIF($1, ''), $2), $2, $3, NULLIF($4, ''))
RETURNING id, name, email, email_verified, avatar_url, status`

	return scanAuthUser(tx.QueryRow(ctx, query, profile.Name, profile.Email, profile.EmailVerified, profile.AvatarURL))
}

func upsertAccount(ctx context.Context, tx pgx.Tx, userID uuid.UUID, profile *oauth.Profile) error {
	const query = `
INSERT INTO accounts (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
ON CONFLICT (provider, provider_user_id)
DO UPDATE SET user_id = EXCLUDED.user_id`

	_, err := tx.Exec(ctx, query, userID, profile.Provider, profile.ProviderUserID)
	return err
}

func scanAuthUser(row pgx.Row) (*AuthUser, error) {
	var user AuthUser
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.EmailVerified,
		&user.AvatarURL,
		&user.Status,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func scanUser(row pgx.Row) (*User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.EmailVerified,
		&user.AvatarURL,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
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
