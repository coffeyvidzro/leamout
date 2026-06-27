package checkout

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin paid checkout completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := r.getSessionByIDForUpdate(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if session.Status == StatusCompleted {
		return tx.Commit(ctx)
	}
	if session.Status != StatusOpen {
		return tx.Commit(ctx)
	}

	session, err = r.completeSession(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if isDunningRenewal(session) {
		if err := r.completeDunningRenewal(ctx, tx, session); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit paid checkout completion: %w", err)
	}
	return nil
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
