package dunning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("dunning record not found")
	ErrActiveTokenExists = errors.New("active dunning token already exists")
	ErrTransitionSkipped = errors.New("dunning transition skipped")
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

func (r *Repository) GetByID(ctx context.Context, attemptID uuid.UUID) (*Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at
FROM dunning_attempts
WHERE id = $1`

	attempt, err := scanAttempt(r.db.QueryRow(ctx, query, attemptID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning attempt by id: %w", err)
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

func (r *Repository) GetReminderDetails(ctx context.Context, userID, subscriptionID uuid.UUID) (*ReminderDetails, error) {
	const query = `
SELECT c.phone, s.current_period_end
FROM subscriptions s
JOIN customers c ON c.user_id = s.user_id AND c.id = s.customer_id
WHERE s.user_id = $1 AND s.id = $2`

	var details ReminderDetails
	err := r.db.QueryRow(ctx, query, userID, subscriptionID).Scan(&details.CustomerPhone, &details.CurrentPeriodEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dunning reminder details: %w", err)
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

func (r *Repository) RevokeAttemptTokens(ctx context.Context, userID, attemptID uuid.UUID) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE dunning_tokens SET revoked_at = COALESCE(revoked_at, NOW()) WHERE user_id = $1 AND dunning_attempt_id = $2 AND revoked_at IS NULL`,
		userID,
		attemptID,
	)
	if err != nil {
		return fmt.Errorf("revoke dunning attempt tokens: %w", err)
	}

	return nil
}

func (r *Repository) RevokeTokenByIDTx(ctx context.Context, tx pgx.Tx, userID, tokenID, attemptID uuid.UUID) error {
	const query = `
UPDATE dunning_tokens
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE user_id = $1
  AND id = $2
  AND dunning_attempt_id = $3
  AND revoked_at IS NULL`

	result, err := tx.Exec(ctx, query, userID, tokenID, attemptID)
	if err != nil {
		return fmt.Errorf("revoke dunning token by id: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) MarkAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	return r.markAttemptSent(ctx, attemptID)
}

func (r *Repository) MarkAttemptClicked(ctx context.Context, attemptID uuid.UUID) error {
	const query = `UPDATE dunning_attempts SET clicked_at = COALESCE(clicked_at, NOW()) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, attemptID)
	return err
}

func (r *Repository) MarkAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	return r.markAttemptPaid(ctx, attemptID)
}

func (r *Repository) MarkAttemptPaidTx(ctx context.Context, tx pgx.Tx, userID, attemptID, subscriptionID, checkoutID uuid.UUID) error {
	if err := setTransitionContext(ctx, tx, "billing", dunningTransitionReasonRenewalPaid, map[string]any{
		"source":              "billing_checkout_completion",
		"checkout_session_id": checkoutID.String(),
	}); err != nil {
		return err
	}

	const query = `
UPDATE dunning_attempts
SET status = 'paid', sent_at = COALESCE(sent_at, NOW()), paid_at = COALESCE(paid_at, NOW())
WHERE user_id = $1
  AND id = $2
  AND subscription_id = $3
  AND status IN ('pending', 'sent')`

	result, err := tx.Exec(ctx, query, userID, attemptID, subscriptionID)
	if err != nil {
		return fmt.Errorf("mark dunning attempt paid: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrTransitionSkipped
	}

	return nil
}

func (r *Repository) MarkAttemptCanceled(ctx context.Context, attemptID uuid.UUID, metadata map[string]any) error {
	return r.markAttemptCanceled(ctx, attemptID, metadata)
}

func (r *Repository) GetConversionMetrics(ctx context.Context, userID uuid.UUID) (*ConversionMetrics, error) {
	const query = `
WITH attempts AS (
	SELECT
		a.id,
		a.status,
		a.expires_at,
		a.sent_at,
		a.clicked_at,
		a.paid_at,
		EXISTS (
			SELECT 1
			FROM checkout_sessions cs
			WHERE cs.user_id = a.user_id
			  AND cs.source = 'dunning'
			  AND cs.metadata->>'dunning_attempt_id' = a.id::text
		) AS checkout_started
	FROM dunning_attempts a
	WHERE a.user_id = $1
)
SELECT
	COUNT(*) FILTER (WHERE sent_at IS NOT NULL OR status IN ('sent', 'paid')) AS sent,
	COUNT(*) FILTER (WHERE clicked_at IS NOT NULL) AS clicked,
	COUNT(*) FILTER (WHERE checkout_started) AS checkout_started,
	COUNT(*) FILTER (WHERE status = 'paid' OR paid_at IS NOT NULL) AS paid,
	COUNT(*) FILTER (WHERE status = 'canceled') AS failed,
	COUNT(*) FILTER (WHERE status = 'expired' OR (status IN ('pending', 'sent') AND expires_at <= NOW())) AS expired
FROM attempts`

	metrics := &ConversionMetrics{}
	if err := r.db.QueryRow(ctx, query, userID).Scan(
		&metrics.Sent,
		&metrics.Clicked,
		&metrics.CheckoutStarted,
		&metrics.Paid,
		&metrics.Failed,
		&metrics.Expired,
	); err != nil {
		return nil, fmt.Errorf("get dunning conversion metrics: %w", err)
	}

	return metrics, nil
}

func (r *Repository) RecordReminderJobFailure(ctx context.Context, params RecordReminderJobFailureParams) (*ReminderJobFailure, error) {
	metadata, err := encodeJSON(defaultMetadata(params.Metadata))
	if err != nil {
		return nil, err
	}

	params.ErrorType = strings.TrimSpace(params.ErrorType)
	if params.ErrorType == "" {
		params.ErrorType = "unknown"
	}
	params.ErrorMessage = strings.TrimSpace(params.ErrorMessage)
	if params.ErrorMessage == "" {
		params.ErrorMessage = "unknown reminder job failure"
	}
	if params.Status == "" {
		params.Status = ReminderJobFailureStatusRetryScheduled
	}

	const query = `
WITH previous_failures AS (
	SELECT COUNT(*) AS count
	FROM dunning_reminder_job_failures
	WHERE user_id = $1
	  AND subscription_id = $2
	  AND customer_id = $3
	  AND current_period_end = $4
), inserted AS (
	INSERT INTO dunning_reminder_job_failures (
		user_id,
		subscription_id,
		customer_id,
		dunning_attempt_id,
		current_period_end,
		failure_number,
		status,
		error_type,
		error_message,
		retryable,
		metadata
	)
	SELECT $1, $2, $3, $5, $4, count + 1, $6, $7, $8, $9, $10
	FROM previous_failures
	RETURNING id, user_id, subscription_id, customer_id, dunning_attempt_id, current_period_end,
		failure_number, status, error_type, error_message, retryable, metadata, created_at
)
SELECT id, user_id, subscription_id, customer_id, dunning_attempt_id, current_period_end,
	failure_number, status, error_type, error_message, retryable, metadata, created_at
FROM inserted`

	failure, err := scanReminderJobFailure(r.db.QueryRow(
		ctx,
		query,
		params.UserID,
		params.SubscriptionID,
		params.CustomerID,
		params.CurrentPeriodEnd,
		params.AttemptID,
		params.Status,
		params.ErrorType,
		params.ErrorMessage,
		params.Retryable,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("record dunning reminder job failure: %w", err)
	}

	return failure, nil
}

func (r *Repository) ListReminderJobFailures(ctx context.Context, userID uuid.UUID) ([]ReminderJobFailure, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, dunning_attempt_id, current_period_end,
	failure_number, status, error_type, error_message, retryable, metadata, created_at
FROM dunning_reminder_job_failures
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 100`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list dunning reminder job failures: %w", err)
	}
	defer rows.Close()

	failures := make([]ReminderJobFailure, 0)
	for rows.Next() {
		failure, err := scanReminderJobFailure(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dunning reminder job failure: %w", err)
		}
		failures = append(failures, *failure)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dunning reminder job failures: %w", err)
	}

	return failures, nil
}

func (r *Repository) ListAttemptTransitions(ctx context.Context, userID, attemptID uuid.UUID) ([]AttemptTransition, error) {
	const query = `
SELECT id, user_id, dunning_attempt_id, actor, reason, previous_status, next_status, metadata, created_at
FROM dunning_attempt_transitions
WHERE user_id = $1
  AND dunning_attempt_id = $2
ORDER BY created_at ASC, id ASC`

	rows, err := r.db.Query(ctx, query, userID, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list dunning attempt transitions: %w", err)
	}
	defer rows.Close()

	transitions := make([]AttemptTransition, 0)
	for rows.Next() {
		transition, err := scanAttemptTransition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dunning attempt transition: %w", err)
		}
		transitions = append(transitions, *transition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dunning attempt transitions: %w", err)
	}

	return transitions, nil
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

func (r *Repository) markAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	return r.transitionAttemptStatus(ctx, attemptID, AttemptStatusSent, []AttemptStatus{AttemptStatusPending}, dunningTransitionActorWorker, dunningTransitionReasonReminderSent, map[string]any{
		"source": "dunning_reminder_worker",
	})
}

func (r *Repository) markAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	return r.transitionAttemptStatus(ctx, attemptID, AttemptStatusPaid, []AttemptStatus{AttemptStatusPending, AttemptStatusSent}, "system", dunningTransitionReasonRenewalPaid, map[string]any{
		"source": "dunning_service",
	})
}

func (r *Repository) markAttemptCanceled(ctx context.Context, attemptID uuid.UUID, metadata map[string]any) error {
	return r.transitionAttemptStatus(ctx, attemptID, AttemptStatusCanceled, []AttemptStatus{AttemptStatusPending, AttemptStatusSent}, dunningTransitionActorWorker, dunningTransitionReasonReminderJobFailed, metadata)
}

func (r *Repository) transitionAttemptStatus(ctx context.Context, attemptID uuid.UUID, next AttemptStatus, allowedPrevious []AttemptStatus, actor, reason string, metadata map[string]any) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dunning transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	attempt, err := r.getAttemptByIDForUpdate(ctx, tx, attemptID)
	if err != nil {
		return err
	}
	if attempt.Status == next {
		return tx.Commit(ctx)
	}
	if !statusIn(attempt.Status, allowedPrevious) {
		return ErrTransitionSkipped
	}

	if err := setTransitionContext(ctx, tx, strings.TrimSpace(actor), strings.TrimSpace(reason), metadata); err != nil {
		return err
	}

	updated, err := r.updateAttemptStatus(ctx, tx, attemptID, next)
	if err != nil {
		return err
	}
	if updated.Status != next {
		return ErrTransitionSkipped
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit dunning transition: %w", err)
	}
	return nil
}

func (r *Repository) getAttemptByIDForUpdate(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID) (*Attempt, error) {
	const query = `
SELECT id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at
FROM dunning_attempts
WHERE id = $1
FOR UPDATE`

	attempt, err := scanAttempt(tx.QueryRow(ctx, query, attemptID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock dunning attempt: %w", err)
	}
	return attempt, nil
}

func (r *Repository) updateAttemptStatus(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID, next AttemptStatus) (*Attempt, error) {
	query := `
UPDATE dunning_attempts
SET status = $2,
	updated_at = NOW()`

	switch next {
	case AttemptStatusSent:
		query += `,
	sent_at = COALESCE(sent_at, NOW())`
	case AttemptStatusPaid:
		query += `,
	sent_at = COALESCE(sent_at, NOW()),
	paid_at = COALESCE(paid_at, NOW())`
	case AttemptStatusCanceled:
		query += `,
	canceled_at = COALESCE(canceled_at, NOW())`
	}

	query += `
WHERE id = $1
RETURNING id, user_id, subscription_id, customer_id, status, reason, period_end, expires_at,
	sent_at, clicked_at, paid_at, canceled_at, metadata, created_at, updated_at`

	attempt, err := scanAttempt(tx.QueryRow(ctx, query, attemptID, next))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTransitionSkipped
	}
	if err != nil {
		return nil, fmt.Errorf("update dunning attempt status: %w", err)
	}
	return attempt, nil
}

func setTransitionContext(ctx context.Context, tx pgx.Tx, actor, reason string, metadata map[string]any) error {
	if actor == "" {
		actor = "system"
	}
	if reason == "" {
		reason = "status_update"
	}
	metadataBytes, err := encodeJSON(defaultMetadata(metadata))
	if err != nil {
		return err
	}

	const query = `
SELECT
	set_config('leamout.dunning_transition_actor', $1, true),
	set_config('leamout.dunning_transition_reason', $2, true),
	set_config('leamout.dunning_transition_metadata', $3, true)`

	if _, err := tx.Exec(ctx, query, actor, reason, string(metadataBytes)); err != nil {
		return fmt.Errorf("set dunning transition context: %w", err)
	}
	return nil
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

func scanReminderJobFailure(row pgx.Row) (*ReminderJobFailure, error) {
	var failure ReminderJobFailure
	var metadataBytes []byte

	if err := row.Scan(
		&failure.ID,
		&failure.UserID,
		&failure.SubscriptionID,
		&failure.CustomerID,
		&failure.AttemptID,
		&failure.CurrentPeriodEnd,
		&failure.FailureNumber,
		&failure.Status,
		&failure.ErrorType,
		&failure.ErrorMessage,
		&failure.Retryable,
		&metadataBytes,
		&failure.CreatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &failure.Metadata); err != nil {
			return nil, fmt.Errorf("decode dunning reminder job failure metadata: %w", err)
		}
	}
	if failure.Metadata == nil {
		failure.Metadata = map[string]any{}
	}

	return &failure, nil
}

func scanAttemptTransition(row pgx.Row) (*AttemptTransition, error) {
	var transition AttemptTransition
	var metadataBytes []byte

	if err := row.Scan(
		&transition.ID,
		&transition.UserID,
		&transition.AttemptID,
		&transition.Actor,
		&transition.Reason,
		&transition.PreviousStatus,
		&transition.NextStatus,
		&metadataBytes,
		&transition.CreatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &transition.Metadata); err != nil {
			return nil, fmt.Errorf("decode dunning transition metadata: %w", err)
		}
	}
	if transition.Metadata == nil {
		transition.Metadata = map[string]any{}
	}

	return &transition, nil
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

func statusIn(status AttemptStatus, statuses []AttemptStatus) bool {
	for _, candidate := range statuses {
		if status == candidate {
			return true
		}
	}
	return false
}
