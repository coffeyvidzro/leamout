package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("payment not found")

func (r *Repository) UpdateFromProvider(ctx context.Context, params UpdateFromProviderParams) (*Payment, error) {
	metadata, err := encodeAnyMap(params.Metadata)
	if err != nil {
		return nil, err
	}

	const query = `
UPDATE payments
SET status = $3,
	provider_reference = COALESCE(NULLIF($4, ''), provider_reference),
	metadata = metadata || $5::jsonb
WHERE external_id = $1 AND provider = $2
RETURNING id, user_id, checkout_id, customer_id, external_id, provider, provider_reference, status,
	currency, amount, fee_amount, net_amount, metadata, created_at, updated_at`

	item, err := scanPayment(r.db.QueryRow(ctx, query, strings.TrimSpace(params.ExternalID), strings.ToLower(strings.TrimSpace(params.Provider)), params.Status, strings.TrimSpace(params.ProviderReference), metadata))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update payment from provider: %w", err)
	}
	return item, nil
}
