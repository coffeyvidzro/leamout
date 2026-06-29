package events

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

var ErrInvalidEvent = errors.New("invalid domain event")

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Publish(ctx context.Context, params PublishParams) (*Event, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("event repository is not configured")
	}
	params.Name = Name(strings.TrimSpace(string(params.Name)))
	params.AggregateType = strings.TrimSpace(params.AggregateType)
	if params.Name == "" || params.AggregateType == "" || params.AggregateID == uuid.Nil {
		return nil, ErrInvalidEvent
	}
	if params.AvailableAt.IsZero() {
		params.AvailableAt = time.Now().UTC()
	}
	payload, err := encodeJSON(params.Payload)
	if err != nil {
		return nil, err
	}
	metadata, err := encodeJSON(params.Metadata)
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO domain_events (user_id, name, aggregate_type, aggregate_id, idempotency_key, payload, metadata, available_at)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8)
ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO UPDATE
SET updated_at = domain_events.updated_at
RETURNING id, user_id, name, aggregate_type, aggregate_id, idempotency_key, payload, metadata,
	status, attempts, available_at, published_at, last_error, created_at, updated_at`
	return scanEvent(r.db.QueryRow(ctx, query, params.UserID, params.Name, params.AggregateType, params.AggregateID, strings.TrimSpace(params.IdempotencyKey), payload, metadata, params.AvailableAt))
}

func (r *Repository) ClaimReady(ctx context.Context, limit int) ([]Event, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("event repository is not configured")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	const query = `
WITH claimed AS (
	SELECT id
	FROM domain_events
	WHERE status IN ('pending', 'failed') AND available_at <= NOW()
	ORDER BY available_at ASC, created_at ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE domain_events e
SET status = 'processing', attempts = attempts + 1, last_error = NULL
FROM claimed
WHERE e.id = claimed.id
RETURNING e.id, e.user_id, e.name, e.aggregate_type, e.aggregate_id, e.idempotency_key, e.payload, e.metadata,
	e.status, e.attempts, e.available_at, e.published_at, e.last_error, e.created_at, e.updated_at`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("claim ready domain events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	return events, rows.Err()
}

func (r *Repository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE domain_events SET status='published', published_at=NOW(), last_error=NULL WHERE id=$1`, id)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	_, err := r.db.Exec(ctx, `UPDATE domain_events SET status='failed', attempts=attempts+1, last_error=NULLIF($2, '') WHERE id=$1`, id, msg)
	return err
}

func scanEvent(row pgx.Row) (*Event, error) {
	var event Event
	var payload, metadata []byte
	if err := row.Scan(&event.ID, &event.UserID, &event.Name, &event.AggregateType, &event.AggregateID, &event.IdempotencyKey, &payload, &metadata, &event.Status, &event.Attempts, &event.AvailableAt, &event.PublishedAt, &event.LastError, &event.CreatedAt, &event.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &event.Payload); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
		return nil, err
	}
	return &event, nil
}

func encodeJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}
