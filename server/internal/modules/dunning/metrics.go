package dunning

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Service) GetConversionMetrics(ctx context.Context, userID uuid.UUID) (*ConversionMetrics, error) {
	return s.repository.GetConversionMetrics(ctx, userID)
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
