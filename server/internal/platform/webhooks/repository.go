package webhooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidWebhook = errors.New("invalid webhook")

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Enqueue(ctx context.Context, params EnqueueParams) ([]Delivery, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("webhook repository is not configured")
	}
	params.EventType = EventType(strings.TrimSpace(string(params.EventType)))
	params.AggregateType = strings.TrimSpace(params.AggregateType)
	if params.UserID == uuid.Nil || params.EventType == "" || params.AggregateType == "" || params.AggregateID == uuid.Nil {
		return nil, ErrInvalidWebhook
	}
	if params.NextAttemptAt.IsZero() {
		params.NextAttemptAt = time.Now().UTC()
	}
	payload, err := encodeJSON(params.Payload)
	if err != nil {
		return nil, err
	}
	eventID := uuid.New()

	const query = `
WITH matching_endpoints AS (
	SELECT id
	FROM webhook_endpoints
	WHERE user_id = $1
	  AND enabled = TRUE
	  AND ($2 = ANY(event_types) OR '*' = ANY(event_types))
), inserted AS (
	INSERT INTO webhook_deliveries (
		endpoint_id, user_id, event_id, event_type, aggregate_type, aggregate_id, idempotency_key, payload, next_attempt_at
	)
	SELECT id, $1, $3, $2, $4, $5, NULLIF($6, ''), $7, $8
	FROM matching_endpoints
	ON CONFLICT (endpoint_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
	RETURNING id, endpoint_id, user_id, event_id, event_type, aggregate_type, aggregate_id, idempotency_key, payload,
		status, attempts, next_attempt_at, delivered_at, last_status_code, last_error, created_at, updated_at
)
SELECT * FROM inserted
ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, params.UserID, params.EventType, eventID, params.AggregateType, params.AggregateID, strings.TrimSpace(params.IdempotencyKey), payload, params.NextAttemptAt)
	if err != nil {
		return nil, fmt.Errorf("enqueue webhook deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]Delivery, 0)
	for rows.Next() {
		delivery, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, *delivery)
	}
	return deliveries, rows.Err()
}

func (r *Repository) ClaimReady(ctx context.Context, limit int) ([]Delivery, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("webhook repository is not configured")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const query = `
WITH claimed AS (
	SELECT id
	FROM webhook_deliveries
	WHERE status IN ('pending', 'failed') AND next_attempt_at <= NOW()
	ORDER BY next_attempt_at ASC, created_at ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE webhook_deliveries d
SET status = 'processing', attempts = attempts + 1, last_error = NULL
FROM claimed
WHERE d.id = claimed.id
RETURNING d.id, d.endpoint_id, d.user_id, d.event_id, d.event_type, d.aggregate_type, d.aggregate_id, d.idempotency_key, d.payload,
	d.status, d.attempts, d.next_attempt_at, d.delivered_at, d.last_status_code, d.last_error, d.created_at, d.updated_at`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("claim ready webhook deliveries: %w", err)
	}
	defer rows.Close()
	deliveries := make([]Delivery, 0)
	for rows.Next() {
		delivery, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, *delivery)
	}
	return deliveries, rows.Err()
}

func (r *Repository) MarkDelivered(ctx context.Context, id uuid.UUID, statusCode int) error {
	_, err := r.db.Exec(ctx, `UPDATE webhook_deliveries SET status='delivered', delivered_at=NOW(), last_status_code=$2, last_error=NULL WHERE id=$1`, id, statusCode)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, statusCode int, nextAttemptAt time.Time, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now().UTC().Add(5 * time.Minute)
	}
	_, err := r.db.Exec(ctx, `UPDATE webhook_deliveries SET status='failed', next_attempt_at=$2, last_status_code=NULLIF($3, 0), last_error=NULLIF($4, '') WHERE id=$1`, id, nextAttemptAt, statusCode, msg)
	return err
}

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func scanDelivery(row pgx.Row) (*Delivery, error) {
	var delivery Delivery
	var payload []byte
	if err := row.Scan(&delivery.ID, &delivery.EndpointID, &delivery.UserID, &delivery.EventID, &delivery.EventType, &delivery.AggregateType, &delivery.AggregateID, &delivery.IdempotencyKey, &payload, &delivery.Status, &delivery.Attempts, &delivery.NextAttemptAt, &delivery.DeliveredAt, &delivery.LastStatusCode, &delivery.LastError, &delivery.CreatedAt, &delivery.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &delivery.Payload); err != nil {
		return nil, err
	}
	return &delivery, nil
}

func encodeJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}
