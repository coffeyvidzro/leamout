package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) Create(ctx context.Context, params CreateParams) (*Payment, error) {
	metadata, err := encodeAnyMap(params.Metadata)
	if err != nil {
		return nil, err
	}
	if params.Status == "" {
		params.Status = StatusPending
	}

	const query = `
INSERT INTO payments (user_id, checkout_id, customer_id, external_id, provider, status, currency, amount, metadata)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9)
RETURNING id, user_id, checkout_id, customer_id, external_id, provider, provider_reference, status,
	currency, amount, fee_amount, net_amount, metadata, created_at, updated_at`

	item, err := scanPayment(r.db.QueryRow(ctx, query,
		params.UserID,
		params.CheckoutID,
		params.CustomerID,
		strings.TrimSpace(params.ExternalID),
		strings.ToLower(strings.TrimSpace(params.Provider)),
		params.Status,
		strings.ToUpper(strings.TrimSpace(params.Currency)),
		params.Amount,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	return item, nil
}

func scanPayment(row pgx.Row) (*Payment, error) {
	var item Payment
	var metadata []byte
	if err := row.Scan(&item.ID, &item.UserID, &item.CheckoutID, &item.CustomerID, &item.ExternalID, &item.Provider, &item.ProviderReference, &item.Status, &item.Currency, &item.Amount, &item.FeeAmount, &item.NetAmount, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	return &item, nil
}
