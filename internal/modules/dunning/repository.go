package dunning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) ListSubscriptionsDueForRenewal(ctx context.Context, start, end time.Time) ([]ExpiringSubscription, error) {
	const query = `
SELECT s.user_id, s.id, s.customer_id, s.current_period_end
FROM subscriptions s
WHERE s.status = 'active'
  AND s.cancel_at_period_end = FALSE
  AND s.customer_id IS NOT NULL
  AND s.current_period_end >= $1
  AND s.current_period_end <= $2
  AND NOT EXISTS (
	SELECT 1
	FROM dunning_attempts a
	WHERE a.user_id = s.user_id
	  AND a.subscription_id = s.id
	  AND a.reason = 'renewal_due'
	  AND a.period_end = s.current_period_end
	  AND a.status IN ('pending', 'sent')
  )
ORDER BY s.current_period_end ASC`

	rows, err := r.db.Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions due for renewal: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]ExpiringSubscription, 0)
	for rows.Next() {
		var subscription ExpiringSubscription
		if err := rows.Scan(
			&subscription.UserID,
			&subscription.SubscriptionID,
			&subscription.CustomerID,
			&subscription.CurrentPeriodEnd,
		); err != nil {
			return nil, fmt.Errorf("scan subscription due for renewal: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriptions due for renewal: %w", err)
	}

	return subscriptions, nil
}

func (r *Repository) GetNotificationDetails(ctx context.Context, userID, attemptID uuid.UUID) (*NotificationDetails, error) {
	const query = `
SELECT c.phone
FROM dunning_attempts a
JOIN customers c ON c.user_id = a.user_id AND c.id = a.customer_id
WHERE a.user_id = $1 AND a.id = $2`

	var details NotificationDetails
	err := r.db.QueryRow(ctx, query, userID, attemptID).Scan(&details.Phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning notification details: %w", err)
	}

	return &details, nil
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

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at
FROM dunning_attempts
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list dunning attempts: %w", err)
	}
	defer rows.Close()

	attempts := make([]Attempt, 0)
	for rows.Next() {
		attempt, err := scanAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dunning attempt: %w", err)
		}
		attempts = append(attempts, *attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dunning attempts: %w", err)
	}

	return attempts, nil
}

func (r *Repository) Get(ctx context.Context, userID, attemptID uuid.UUID) (*Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at
FROM dunning_attempts
WHERE user_id = $1 AND id = $2`

	attempt, err := scanAttempt(r.db.QueryRow(ctx, query, userID, attemptID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning attempt: %w", err)
	}

	return attempt, nil
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

func (r *Repository) GetCheckoutDetails(ctx context.Context, userID, attemptID uuid.UUID) (*CheckoutDetails, error) {
	const query = `
SELECT p.unit_amount, p.currency
FROM dunning_attempts a
JOIN subscriptions s ON s.user_id = a.user_id AND s.id = a.subscription_id
JOIN prices p ON p.user_id = s.user_id AND p.id = s.price_id
WHERE a.user_id = $1 AND a.id = $2`

	var details CheckoutDetails
	err := r.db.QueryRow(ctx, query, userID, attemptID).Scan(&details.Amount, &details.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning checkout details: %w", err)
	}

	return &details, nil
}

func (r *Repository) GetByTokenHash(ctx context.Context, tokenHash string) (*TokenWithAttempt, error) {
	const query = `
SELECT
	t.id, t.user_id, t.dunning_attempt_id, t.token_hash, t.expires_at, t.revoked_at,
	t.last_used_at, t.created_at, t.updated_at,
	a.id, a.user_id, a.subscription_id, a.customer_id, a.status, a.reason, a.period_end, a.expires_at,
	a.sent_at, a.clicked_at, a.paid_at, a.canceled_at, a.metadata, a.created_at, a.updated_at
FROM dunning_tokens t
JOIN dunning_attempts a ON a.user_id = t.user_id AND a.id = t.dunning_attempt_id
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

func (r *Repository) RecordTokenUse(ctx context.Context, tokenHash string) (*TokenWithAttempt, error) {
	const query = `
WITH touched AS (
	UPDATE dunning_tokens
	SET last_used_at = NOW()
	WHERE token_hash = $1
	  AND revoked_at IS NULL
	  AND expires_at > NOW()
	RETURNING id, user_id, dunning_attempt_id, token_hash, expires_at, revoked_at, last_used_at, created_at, updated_at
)
SELECT
	t.id, t.user_id, t.dunning_attempt_id, t.token_hash, t.expires_at, t.revoked_at,
	t.last_used_at, t.created_at, t.updated_at,
	a.id, a.user_id, a.subscription_id, a.customer_id, a.status, a.reason, a.period_end, a.expires_at,
	a.sent_at, a.clicked_at, a.paid_at, a.canceled_at, a.metadata, a.created_at, a.updated_at
FROM touched t
JOIN dunning_attempts a ON a.user_id = t.user_id AND a.id = t.dunning_attempt_id`

	result, err := scanTokenWithAttempt(r.db.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("record dunning token use: %w", err)
	}

	return result, nil
}

func (r *Repository) RevokeToken(ctx context.Context, tokenHash string) error {
	result, err := r.db.Exec(ctx, `UPDATE dunning_tokens SET revoked_at = COALESCE(revoked_at, NOW()) WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke dunning token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) MarkAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET status = 'sent', sent_at = COALESCE(sent_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) MarkAttemptClicked(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET clicked_at = COALESCE(clicked_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) MarkAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET status = 'paid', sent_at = COALESCE(sent_at, NOW()), paid_at = COALESCE(paid_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) findReusableAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at
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
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at`

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
RETURNING id, user_id, dunning_attempt_id, token_hash, expires_at, revoked_at, last_used_at, created_at, updated_at`

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
		&attempt.ClickedAt,
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
		&token.RevokedAt,
		&token.LastUsedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
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
		&result.Token.RevokedAt,
		&result.Token.LastUsedAt,
		&result.Token.CreatedAt,
		&result.Token.UpdatedAt,
		&result.Attempt.ID,
		&result.Attempt.UserID,
		&result.Attempt.SubscriptionID,
		&result.Attempt.CustomerID,
		&result.Attempt.Status,
		&result.Attempt.Reason,
		&result.Attempt.PeriodEnd,
		&result.Attempt.ExpiresAt,
		&result.Attempt.SentAt,
		&result.Attempt.ClickedAt,
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
