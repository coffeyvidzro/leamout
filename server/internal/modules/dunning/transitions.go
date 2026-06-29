package dunning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	dunningTransitionActorWorker   = "worker"
	dunningTransitionActorCheckout = "checkout"

	dunningTransitionReasonReminderSent = "reminder_sent"
	dunningTransitionReasonRenewalPaid  = "renewal_paid"
)

func (s *Service) ListAttemptTransitions(ctx context.Context, userID, attemptID uuid.UUID) ([]AttemptTransition, error) {
	return s.repository.ListAttemptTransitions(ctx, userID, attemptID)
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

	updated, err := r.updateAttemptStatus(ctx, tx, attemptID, next)
	if err != nil {
		return err
	}
	if updated.Status != next {
		return ErrTransitionSkipped
	}

	if err := r.insertAttemptTransition(ctx, tx, attempt.UserID, attempt.ID, strings.TrimSpace(actor), strings.TrimSpace(reason), attempt.Status, updated.Status, metadata); err != nil {
		return err
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

func (r *Repository) insertAttemptTransition(ctx context.Context, tx pgx.Tx, userID, attemptID uuid.UUID, actor, reason string, previous, next AttemptStatus, metadata map[string]any) error {
	metadataBytes, err := encodeJSON(defaultMetadata(metadata))
	if err != nil {
		return err
	}

	const query = `
INSERT INTO dunning_attempt_transitions (
	user_id,
	dunning_attempt_id,
	actor,
	reason,
	previous_status,
	next_status,
	metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := tx.Exec(ctx, query, userID, attemptID, actor, reason, previous, next, metadataBytes); err != nil {
		return fmt.Errorf("insert dunning attempt transition: %w", err)
	}
	return nil
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

func statusIn(status AttemptStatus, statuses []AttemptStatus) bool {
	for _, candidate := range statuses {
		if status == candidate {
			return true
		}
	}
	return false
}
