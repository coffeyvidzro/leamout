package dunning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) RecordReminderJobFailure(ctx context.Context, params RecordReminderJobFailureParams) (*ReminderJobFailure, error) {
	return s.repository.RecordReminderJobFailure(ctx, params)
}

func (s *Service) ListReminderJobFailures(ctx context.Context, userID uuid.UUID) ([]ReminderJobFailure, error) {
	return s.repository.ListReminderJobFailures(ctx, userID)
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
