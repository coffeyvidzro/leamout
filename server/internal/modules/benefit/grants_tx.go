package benefit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GrantSubscriptionBenefitsTx(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	const query = `
INSERT INTO benefit_grants (
	user_id,
	benefit_id,
	customer_id,
	product_id,
	subscription_id,
	source_type,
	source_id,
	status,
	starts_at,
	ends_at,
	properties,
	metadata
)
SELECT
	s.user_id,
	b.id,
	COALESCE(s.customer_id, $3),
	p.product_id,
	s.id,
	'subscription',
	s.id,
	'active',
	s.current_period_start,
	s.current_period_end,
	b.properties,
	jsonb_build_object(
		'source', 'billing_checkout_completion',
		'checkout_session_id', $4::text,
		'subscription_id', s.id::text,
		'product_id', p.product_id::text
	)
FROM subscriptions s
JOIN prices p
  ON p.user_id = s.user_id
 AND p.id = s.price_id
JOIN product_benefits pb
  ON pb.user_id = s.user_id
 AND pb.product_id = p.product_id
JOIN benefits b
  ON b.user_id = pb.user_id
 AND b.id = pb.benefit_id
 AND b.archived_at IS NULL
WHERE s.user_id = $1
  AND s.id = $2
  AND COALESCE(s.customer_id, $3) IS NOT NULL
ON CONFLICT (user_id, customer_id, benefit_id, source_type, source_id)
DO UPDATE SET
	product_id = EXCLUDED.product_id,
	subscription_id = EXCLUDED.subscription_id,
	status = 'active',
	starts_at = EXCLUDED.starts_at,
	ends_at = EXCLUDED.ends_at,
	properties = EXCLUDED.properties,
	metadata = benefit_grants.metadata || EXCLUDED.metadata,
	revoked_at = NULL,
	updated_at = NOW()`

	if _, err := tx.Exec(ctx, query, userID, subscriptionID, fallbackCustomerID, checkoutID); err != nil {
		return fmt.Errorf("grant subscription benefits: %w", err)
	}

	return nil
}
