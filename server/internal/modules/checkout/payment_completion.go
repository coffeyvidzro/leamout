package checkout

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
