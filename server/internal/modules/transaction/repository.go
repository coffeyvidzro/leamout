package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (*Transaction, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return nil, err
	}
	if params.OccurredAt.IsZero() {
		params.OccurredAt = time.Now().UTC()
	}

	const query = `
INSERT INTO transactions (
	user_id, payment_id, checkout_id, external_id, type, status, currency, amount, occurred_at, metadata
)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id, external_id) WHERE external_id IS NOT NULL DO UPDATE SET
	status = EXCLUDED.status,
	metadata = transactions.metadata || EXCLUDED.metadata
RETURNING id, user_id, payment_id, checkout_id, external_id, type, status, currency, amount, occurred_at, metadata, created_at`

	tx, err := scanTransaction(r.db.QueryRow(
		ctx,
		query,
		params.UserID,
		params.PaymentID,
		params.CheckoutID,
		strings.TrimSpace(params.ExternalID),
		params.Type,
		params.Status,
		strings.ToUpper(strings.TrimSpace(params.Currency)),
		params.Amount,
		params.OccurredAt,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	return tx, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) ([]Transaction, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	query := `
SELECT id, user_id, payment_id, checkout_id, external_id, type, status, currency, amount, occurred_at, metadata, created_at
FROM transactions
WHERE user_id = $1`
	args := []any{params.UserID}
	if params.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, params.Type)
	}
	query += fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	items := make([]Transaction, 0)
	for rows.Next() {
		item, err := scanTransaction(rows)
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

func scanTransaction(row pgx.Row) (*Transaction, error) {
	var item Transaction
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.PaymentID,
		&item.CheckoutID,
		&item.ExternalID,
		&item.Type,
		&item.Status,
		&item.Currency,
		&item.Amount,
		&item.OccurredAt,
		&metadata,
		&item.CreatedAt,
	); err != nil {
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

func encodeMetadata(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}
