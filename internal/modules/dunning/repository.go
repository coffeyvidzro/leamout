package dunning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("dunning record not found")
	ErrActiveTokenExists = errors.New("active dunning token already exists")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrReuseAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	attempt, err := r.findReusableAttempt(ctx, params)
	if err == nil {
		return attempt, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	created, err := r.createAttempt(ctx, params)
	if err == nil {
		return created, nil
	}
	if isUniqueViolation(err) {
		return r.findReusableAttempt(ctx, params)
	}

	return nil, err
}

func (r *Repository) GetAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	return r.findReusableAttempt(ctx, params)
}

func (r *Repository) CreateToken(ctx context.Context, params CreateTokenParams) (*Token, error) {
	token, err := r.createToken(ctx, params)
	if err == nil {
		return token, nil
	}
	if isUniqueViolation(err) {
		return nil, ErrActiveTokenExists
	}

	return nil, err
}

func (r *Repository) GetByTokenHash(ctx context.Context, tokenHash string) (*TokenWithAttempt, error) {
	const query = `
SELECT
	t.id, t.user_id, t.dunning_attempt_id, t.token_hash, t.expires_at, t.used_at, t.created_at,
	a.id, a.user_id, a.subscription_id, a.customer_id, a.status, a.reason, a.period_end, a.expires_at,
	a.sent_at, a.paid_at, a.canceled_at, a.metadata, a.created_at, a.updated_at
FROM dunning_tokens t
JOIN dunning_attempts a ON a.id = t.dunning_attempt_id
WHERE t.token_hash = $1`

	result, err := scanTokenWithAttempt(r.db.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning token: %w", err)
	}

	return result, nil
}

func (r *Repository) ConsumeToken(ctx context.Context, tokenHash string) (*TokenWithAttempt, error) {
	const query = `
WITH consumed AS (
	UPDATE dunning_tokens
	SET used_at = NOW()
	WHERE token_hash = $1
	  AND used_at IS NULL
	  AND expires_at > NOW()
	RETURNING id, user_id, dunning_attempt_id, token_hash, expires_at, used_at, created_at
)
SELECT
	c.id, c.user_id, c.dunning_attempt_id, c.token_hash, c.expires_at, c.used_at, c.created_at,
	a.id, a.user_id, a.subscription_id, a.customer_id, a.status, a.reason, a.period_end, a.expires_at,
	a.sent_at, a.paid_at, a.canceled_at, a.metadata, a.created_at, a.updated_at
FROM consumed c
JOIN dunning_attempts a ON a.id = c.dunning_attempt_id`

	result, err := scanTokenWithAttempt(r.db.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consume dunning token: %w", err)
	}

	return result, nil
}

func (r *Repository) MarkAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET status = 'sent', sent_at = COALESCE(sent_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) MarkAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET status = 'paid', paid_at = COALESCE(paid_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) findReusableAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, paid_at, canceled_at, metadata, created_at, updated_at
FROM dunning_attempts
WHERE user_id = $1
  AND subscription_id = $2
  AND reason = $3
  AND period_end = $4
  AND status IN ('pending', 'sent')
ORDER BY created_at DESC
LIMIT 1`

	attempt, err := scanAttempt(r.db.QueryRow(ctx, query, params.UserID, params.SubscriptionID, params.Reason, params.PeriodEnd))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find dunning attempt: %w", err)
	}

	return attempt, nil
}

func (r *Repository) createAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	metadata, err := encodeJSON(defaultMetadata(params.Metadata))
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO dunning_attempts (user_id, subscription_id, customer_id, status, reason, period_end, expires_at, metadata)
VALUES ($1, $2, $3, 'pending', $4, $5, $6, $7)
RETURNING id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, paid_at, canceled_at, metadata, created_at, updated_at`

	attempt, err := scanAttempt(r.db.QueryRow(
		ctx,
		query,
		params.UserID,
		params.SubscriptionID,
		params.CustomerID,
		params.Reason,
		params.PeriodEnd,
		params.ExpiresAt,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create dunning attempt: %w", err)
	}

	return attempt, nil
}

func (r *Repository) createToken(ctx context.Context, params CreateTokenParams) (*Token, error) {
	const query = `
INSERT INTO dunning_tokens (user_id, dunning_attempt_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, dunning_attempt_id, token_hash, expires_at, used_at, created_at`

	token, err := scanToken(r.db.QueryRow(ctx, query, params.UserID, params.DunningAttemptID, params.TokenHash, params.ExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("create dunning token: %w", err)
	}

	return token, nil
}

func scanAttempt(row pgx.Row) (*Attempt, error) {
	var attempt Attempt
	var metadataBytes []byte

	if err := row.Scan(
		&attempt.ID,
		&attempt.UserID,
		&attempt.SubscriptionID,
		&attempt.CustomerID,
		&attempt.Status,
		&attempt.Reason,
		&attempt.PeriodEnd,
		&attempt.ExpiresAt,
		&attempt.SentAt,
		&attempt.PaidAt,
		&attempt.CanceledAt,
		&metadataBytes,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &attempt.Metadata); err != nil {
			return nil, fmt.Errorf("decode dunning attempt metadata: %w", err)
		}
	}
	if attempt.Metadata == nil {
		attempt.Metadata = map[string]any{}
	}

	return &attempt, nil
}

func scanToken(row pgx.Row) (*Token, error) {
	var token Token
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.DunningAttemptID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &token, nil
}

func scanTokenWithAttempt(row pgx.Row) (*TokenWithAttempt, error) {
	var result TokenWithAttempt
	var metadataBytes []byte

	if err := row.Scan(
		&result.Token.ID,
		&result.Token.UserID,
		&result.Token.DunningAttemptID,
		&result.Token.TokenHash,
		&result.Token.ExpiresAt,
		&result.Token.UsedAt,
		&result.Token.CreatedAt,
		&result.Attempt.ID,
		&result.Attempt.UserID,
		&result.Attempt.SubscriptionID,
		&result.Attempt.CustomerID,
		&result.Attempt.Status,
		&result.Attempt.Reason,
		&result.Attempt.PeriodEnd,
		&result.Attempt.ExpiresAt,
		&result.Attempt.SentAt,
		&result.Attempt.PaidAt,
		&result.Attempt.CanceledAt,
		&metadataBytes,
		&result.Attempt.CreatedAt,
		&result.Attempt.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &result.Attempt.Metadata); err != nil {
			return nil, fmt.Errorf("decode dunning attempt metadata: %w", err)
		}
	}
	if result.Attempt.Metadata == nil {
		result.Attempt.Metadata = map[string]any{}
	}

	return &result, nil
}

func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	return data, nil
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	return metadata
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
