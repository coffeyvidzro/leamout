package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrPaymentAttemptNotFound = errors.New("checkout payment attempt not found")

func (r *Repository) CreatePaymentAttempt(ctx context.Context, params CreatePaymentAttemptParams) error {
	providerResponse, err := encodeRawJSON(params.ProviderResponse)
	if err != nil {
		return err
	}
	metadata, err := encodeStringMap(params.Metadata)
	if err != nil {
		return err
	}

	status := params.Status
	if status == "" {
		status = PaymentAttemptStatusPending
	}

	const query = `
INSERT INTO checkout_payment_attempts (
	checkout_session_id,
	user_id,
	external_ref,
	provider_id,
	provider_reference,
	status,
	amount,
	currency,
	country,
	payment_method,
	operator,
	customer_phone,
	provider_response,
	metadata
)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10, NULLIF($11, ''), NULLIF($12, ''), $13, $14)
ON CONFLICT (external_ref) DO UPDATE SET
	provider_id = EXCLUDED.provider_id,
	provider_reference = COALESCE(EXCLUDED.provider_reference, checkout_payment_attempts.provider_reference),
	status = EXCLUDED.status,
	provider_response = EXCLUDED.provider_response,
	metadata = checkout_payment_attempts.metadata || EXCLUDED.metadata`

	_, err = r.db.Exec(
		ctx,
		query,
		params.CheckoutSessionID,
		params.UserID,
		strings.TrimSpace(params.ExternalRef),
		strings.ToLower(strings.TrimSpace(params.ProviderID)),
		strings.TrimSpace(params.ProviderReference),
		status,
		params.Amount,
		strings.ToUpper(strings.TrimSpace(params.Currency)),
		strings.ToUpper(strings.TrimSpace(params.Country)),
		strings.ToLower(strings.TrimSpace(params.PaymentMethod)),
		strings.ToLower(strings.TrimSpace(params.Operator)),
		strings.TrimSpace(params.CustomerPhone),
		providerResponse,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("create checkout payment attempt: %w", err)
	}

	return nil
}

func (r *Repository) ApplyPaymentResult(ctx context.Context, params ApplyPaymentResultParams) error {
	providerResponse, err := encodeRawJSON(params.ProviderResponse)
	if err != nil {
		return err
	}
	metadata, err := encodeStringMap(params.Metadata)
	if err != nil {
		return err
	}

	status := params.Status
	if status == "" {
		status = PaymentAttemptStatusUnknown
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin checkout payment result: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	attemptID, sessionID, err := r.findPaymentAttemptForUpdate(ctx, tx, params)
	if err != nil {
		return err
	}

	const updateAttempt = `
UPDATE checkout_payment_attempts
SET status = $2,
	provider_reference = COALESCE(NULLIF($3, ''), provider_reference),
	provider_response = CASE WHEN $4::jsonb = '{}'::jsonb THEN provider_response ELSE $4::jsonb END,
	metadata = metadata || $5::jsonb
WHERE id = $1`

	if _, err := tx.Exec(ctx, updateAttempt, attemptID, status, strings.TrimSpace(params.ProviderReference), providerResponse, metadata); err != nil {
		return fmt.Errorf("update checkout payment attempt: %w", err)
	}

	if status == PaymentAttemptStatusSucceeded {
		session, err := r.getSessionByIDForUpdate(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session.Status == StatusCompleted {
			return tx.Commit(ctx)
		}
		if session.Status != StatusOpen {
			return tx.Commit(ctx)
		}

		session, err = r.completeSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if isDunningRenewal(session) {
			if err := r.completeDunningRenewal(ctx, tx, session); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit checkout payment result: %w", err)
	}

	return nil
}

func (r *Repository) findPaymentAttemptForUpdate(ctx context.Context, tx pgx.Tx, params ApplyPaymentResultParams) (uuid.UUID, uuid.UUID, error) {
	const query = `
SELECT id, checkout_session_id
FROM checkout_payment_attempts
WHERE ($1 <> '' AND external_ref = $1)
   OR ($3 <> '' AND provider_id = $2 AND provider_reference = $3)
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE`

	var attemptID uuid.UUID
	var sessionID uuid.UUID
	err := tx.QueryRow(
		ctx,
		query,
		strings.TrimSpace(params.ExternalRef),
		strings.ToLower(strings.TrimSpace(params.ProviderID)),
		strings.TrimSpace(params.ProviderReference),
	).Scan(&attemptID, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrPaymentAttemptNotFound
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("find checkout payment attempt: %w", err)
	}

	return attemptID, sessionID, nil
}

func (r *Repository) getSessionByIDForUpdate(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE id = $1
FOR UPDATE`

	session, err := scanSession(tx.QueryRow(ctx, query, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock checkout session by id: %w", err)
	}

	return session, nil
}

func encodeRawJSON(raw []byte) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []byte(`{}`), nil
	}
	if json.Valid(raw) {
		return raw, nil
	}

	return json.Marshal(map[string]string{"raw": string(raw)})
}

func encodeStringMap(value map[string]string) ([]byte, error) {
	if len(value) == 0 {
		return []byte(`{}`), nil
	}

	return json.Marshal(value)
}
