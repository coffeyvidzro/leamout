package payment

import (
	"context"
	"fmt"
)

func (r *Repository) List(ctx context.Context, params ListParams) ([]Payment, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	query := `
SELECT id, user_id, checkout_id, customer_id, external_id, provider, provider_reference, status,
	currency, amount, fee_amount, net_amount, metadata, created_at, updated_at
FROM payments
WHERE user_id = $1`
	args := []any{params.UserID}
	if params.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, params.Status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	items := make([]Payment, 0)
	for rows.Next() {
		item, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
