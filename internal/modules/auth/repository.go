package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	CreateUser(ctx context.Context, profile *oauth.Profile) (*User, error)
	FindAccount(ctx context.Context, provider, providerUserID string) (*Account, error)
	CreateAccount(ctx context.Context, userID uuid.UUID, profile *oauth.Profile) (*Account, error)
	CreateSession(ctx context.Context, session CreateSessionParams) (*Session, error)
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	TouchSession(ctx context.Context, id uuid.UUID) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
}

type CreateSessionParams struct {
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

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	const query = `
SELECT id, name, email, email_verified, avatar_url, password_hash, status, created_at, updated_at
FROM users
WHERE email = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = `
SELECT id, name, email, email_verified, avatar_url, password_hash, status, created_at, updated_at
FROM users
WHERE id = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, profile *oauth.Profile) (*User, error) {
	const query = `
INSERT INTO users (name, email, email_verified, avatar_url)
VALUES ($1, $2, $3, NULLIF($4, ''))
RETURNING id, name, email, email_verified, avatar_url, password_hash, status, created_at, updated_at`

	user, err := scanUser(r.pool.QueryRow(ctx, query, profile.Name, profile.Email, profile.EmailVerified, profile.AvatarURL))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) FindAccount(ctx context.Context, provider, providerUserID string) (*Account, error) {
	const query = `
SELECT id, user_id, provider, provider_user_id, created_at, updated_at
FROM accounts
WHERE provider = $1 AND provider_user_id = $2`

	account, err := scanAccount(r.pool.QueryRow(ctx, query, provider, providerUserID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find account: %w", err)
	}

	return account, nil
}

func (r *PostgresRepository) CreateAccount(ctx context.Context, userID uuid.UUID, profile *oauth.Profile) (*Account, error) {
	const query = `
INSERT INTO accounts (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING id, user_id, provider, provider_user_id, created_at, updated_at`

	account, err := scanAccount(r.pool.QueryRow(ctx, query, userID, profile.Provider, profile.ProviderUserID))
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error) {
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

func (r *PostgresRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at, updated_at
FROM sessions
WHERE token_hash = $1`

	session, err := scanSession(r.pool.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session by token hash: %w", err)
	}

	return session, nil
}

func (r *PostgresRepository) TouchSession(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1`

	if _, err := r.pool.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	const query = `
UPDATE sessions
SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL`

	result, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
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
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func scanAccount(row pgx.Row) (*Account, error) {
	var account Account
	if err := row.Scan(
		&account.ID,
		&account.UserID,
		&account.Provider,
		&account.ProviderUserID,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &account, nil
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
