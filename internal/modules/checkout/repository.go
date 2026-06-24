package checkout

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

var ErrNotFound = errors.New("checkout session not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrReuseFromDunning(ctx context.Context, params CreateSessionParams) (*Session, error) {
	session, err := r.findReusable(ctx, params)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	created, err := r.createFromDunning(ctx, params)
	if err == nil {
		return created, nil
	}
	if isUniqueViolation(err) {
		return r.findReusable(ctx, params)
	}

	return nil, err
}

func (r *Repository) Complete(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	const query = `
UPDATE checkout_sessions
SET status = 'completed', completed_at = COALESCE(completed_at, NOW())
WHERE id = $1
  AND status = 'open'
  AND expires_at > NOW()
RETURNING id, user_id, customer_id, subscription_id, dunning_attempt_id, dunning_token_id, status,
	amount, currency, expires_at, completed_at, canceled_at, metadata, created_at, updated_at`

	session, err := scanSession(r.db.QueryRow(ctx, query, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("complete checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) findReusable(ctx context.Context, params CreateSessionParams) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, dunning_attempt_id, dunning_token_id, status,
	amount, currency, expires_at, completed_at, canceled_at, metadata, created_at, updated_at
FROM checkout_sessions
WHERE dunning_token_id = $1
   OR (dunning_attempt_id = $2 AND status = 'open' AND expires_at > NOW())
ORDER BY created_at DESC
LIMIT 1`

	session, err := scanSession(r.db.QueryRow(ctx, query, params.DunningTokenID, params.DunningAttemptID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) createFromDunning(ctx context.Context, params CreateSessionParams) (*Session, error) {
	metadata, err := encodeJSON(defaultMetadata(params.Metadata))
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO checkout_sessions (
	user_id,
	customer_id,
	subscription_id,
	dunning_attempt_id,
	dunning_token_id,
	amount,
	currency,
	expires_at,
	metadata
)
SELECT
	$1,
	$2,
	$3,
	$4,
	$5,
	p.unit_amount,
	p.currency,
	$6,
	$7
FROM subscriptions s
JOIN prices p ON p.id = s.price_id AND p.user_id = s.user_id
WHERE s.user_id = $1
  AND s.id = $3
RETURNING id, user_id, customer_id, subscription_id, dunning_attempt_id, dunning_token_id, status,
	amount, currency, expires_at, completed_at, canceled_at, metadata, created_at, updated_at`

	session, err := scanSession(r.db.QueryRow(
		ctx,
		query,
		params.UserID,
		params.CustomerID,
		params.SubscriptionID,
		params.DunningAttemptID,
		params.DunningTokenID,
		params.ExpiresAt,
		metadata,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return session, nil
}

func scanSession(row pgx.Row) (*Session, error) {
	var session Session
	var metadataBytes []byte

	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.CustomerID,
		&session.SubscriptionID,
		&session.DunningAttemptID,
		&session.DunningTokenID,
		&session.Status,
		&session.Amount,
		&session.Currency,
		&session.ExpiresAt,
		&session.CompletedAt,
		&session.CanceledAt,
		&metadataBytes,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &session.Metadata); err != nil {
			return nil, fmt.Errorf("decode checkout session metadata: %w", err)
		}
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}

	return &session, nil
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
