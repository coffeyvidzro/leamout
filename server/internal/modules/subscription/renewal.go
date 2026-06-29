package subscription

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) RenewPeriodTx(ctx context.Context, tx pgx.Tx, userID, subscriptionID uuid.UUID) error {
	const query = `
UPDATE subscriptions s
SET current_period_start = s.current_period_end,
	current_period_end = CASE p.interval
		WHEN 'day' THEN s.current_period_end + INTERVAL '1 day'
		WHEN 'week' THEN s.current_period_end + INTERVAL '1 week'
		WHEN 'month' THEN s.current_period_end + INTERVAL '1 month'
		WHEN 'year' THEN s.current_period_end + INTERVAL '1 year'
		ELSE s.current_period_end
	END
FROM prices p
WHERE s.user_id = $1
  AND s.id = $2
  AND p.user_id = s.user_id
  AND p.id = s.price_id
  AND p.type = 'recurring'
  AND p.interval IS NOT NULL
  AND s.status = 'active'
  AND s.cancel_at_period_end = FALSE`

	result, err := tx.Exec(ctx, query, userID, subscriptionID)
	if err != nil {
		return fmt.Errorf("renew subscription period: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}

	return nil
}
