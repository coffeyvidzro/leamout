package dunning

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
