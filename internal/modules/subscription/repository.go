package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("subscription not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Subscription, error) {
	metadata, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	status := req.Status
	if status == "" {
		status = StatusActive
	}
	periodStart := req.CurrentPeriodStart
	if periodStart.IsZero() {
		periodStart = time.Now().UTC()
	}

	const query = `
INSERT INTO subscriptions (
	user_id,
	customer_id,
	price_id,
	status,
	current_period_start,
	current_period_end,
	metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, customer_id, price_id, status, current_period_start, current_period_end,
	cancel_at_period_end, canceled_at, ends_at, ended_at, customer_cancellation_reason,
	customer_cancellation_comment, metadata, created_at, updated_at`

	subscription, err := scanSubscription(r.db.QueryRow(
		ctx,
		query,
		userID,
		req.CustomerID,
		req.PriceID,
		status,
		periodStart,
		req.CurrentPeriodEnd,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	return subscription, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	const query = `
SELECT id, user_id, customer_id, price_id, status, current_period_start, current_period_end,
	cancel_at_period_end, canceled_at, ends_at, ended_at, customer_cancellation_reason,
	customer_cancellation_comment, metadata, created_at, updated_at
FROM subscriptions
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]Subscription, 0)
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, *subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriptions: %w", err)
	}

	return subscriptions, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Subscription, error) {
	const query = `
SELECT id, user_id, customer_id, price_id, status, current_period_start, current_period_end,
	cancel_at_period_end, canceled_at, ends_at, ended_at, customer_cancellation_reason,
	customer_cancellation_comment, metadata, created_at, updated_at
FROM subscriptions
WHERE user_id = $1 AND id = $2`

	return r.get(ctx, query, userID, id)
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Subscription, error) {
	query, args, err := buildUpdateQuery([]any{userID, id}, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	return r.get(ctx, query, args...)
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM subscriptions WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) get(ctx context.Context, query string, args ...any) (*Subscription, error) {
	subscription, err := scanSubscription(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	return subscription, nil
}

func buildUpdateQuery(args []any, req UpdateRequest) (string, []any, error) {
	updates := make([]string, 0, 8)

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *req.Status)
	}
	if req.CurrentPeriodEnd != nil {
		updates = append(updates, fmt.Sprintf("current_period_end = $%d", len(args)+1))
		args = append(args, *req.CurrentPeriodEnd)
	}
	if req.CancelAtPeriodEnd != nil {
		updates = append(updates, fmt.Sprintf("cancel_at_period_end = $%d", len(args)+1))
		args = append(args, *req.CancelAtPeriodEnd)
	}
	if req.CanceledAt != nil {
		updates = append(updates, fmt.Sprintf("canceled_at = $%d", len(args)+1))
		args = append(args, *req.CanceledAt)
	}
	if req.EndsAt != nil {
		updates = append(updates, fmt.Sprintf("ends_at = $%d", len(args)+1))
		args = append(args, *req.EndsAt)
	}
	if req.EndedAt != nil {
		updates = append(updates, fmt.Sprintf("ended_at = $%d", len(args)+1))
		args = append(args, *req.EndedAt)
	}
	if req.CustomerCancellationReason != nil {
		updates = append(updates, fmt.Sprintf("customer_cancellation_reason = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.CustomerCancellationReason))
	}
	if req.CustomerCancellationComment != nil {
		updates = append(updates, fmt.Sprintf("customer_cancellation_comment = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.CustomerCancellationComment))
	}
	if req.Metadata != nil {
		metadata, err := encodeJSON(req.Metadata)
		if err != nil {
			return "", nil, err
		}
		updates = append(updates, fmt.Sprintf("metadata = $%d", len(args)+1))
		args = append(args, metadata)
	}
	if len(updates) == 0 {
		return "", args, nil
	}

	query := fmt.Sprintf(`
UPDATE subscriptions
SET %s
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, customer_id, price_id, status, current_period_start, current_period_end,
	cancel_at_period_end, canceled_at, ends_at, ended_at, customer_cancellation_reason,
	customer_cancellation_comment, metadata, created_at, updated_at`, strings.Join(updates, ", "))

	return query, args, nil
}

func scanSubscription(row pgx.Row) (*Subscription, error) {
	var subscription Subscription
	var metadataBytes []byte

	if err := row.Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.CustomerID,
		&subscription.PriceID,
		&subscription.Status,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&subscription.CanceledAt,
		&subscription.EndsAt,
		&subscription.EndedAt,
		&subscription.CustomerCancellationReason,
		&subscription.CustomerCancellationComment,
		&metadataBytes,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("decode subscription metadata: %w", err)
		}
	}
	if subscription.Metadata == nil {
		subscription.Metadata = map[string]any{}
	}

	return &subscription, nil
}

func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	return data, nil
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	return metadata
}
