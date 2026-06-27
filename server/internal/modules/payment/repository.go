package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("payment not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

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

	item, err := scanPayment(r.db.QueryRow(ctx, query,
		strings.TrimSpace(params.ExternalID),
		strings.ToLower(strings.TrimSpace(params.Provider)),
		params.Status,
		strings.TrimSpace(params.ProviderReference),
		metadata,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update payment from provider: %w", err)
	}
	return item, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Payment, error) {
	const query = `
SELECT id, user_id, checkout_id, customer_id, external_id, provider, provider_reference, status,
	currency, amount, fee_amount, net_amount, metadata, created_at, updated_at
FROM payments
WHERE user_id = $1 AND id = $2`

	item, err := scanPayment(r.db.QueryRow(ctx, query, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return item, nil
}

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

func encodeAnyMap(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

func stringMapToAny(src map[string]string) map[string]any {
	out := make(map[string]any, len(src))
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}
