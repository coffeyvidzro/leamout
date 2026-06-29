package usageevent

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

var ErrNotFound = errors.New("usage event not found")

type Repository struct {
	db *pgxpool.Pool
}

type usageEventConsumer func(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, CreateParams) error

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Ingest(ctx context.Context, userID uuid.UUID, events []CreateParams) (*IngestResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin usage event ingest: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	response := &IngestResponse{}
	for _, event := range events {
		if err := r.resolveEventCustomer(ctx, tx, userID, &event); err != nil {
			return nil, err
		}
		if err := r.validateReferences(ctx, tx, userID, event); err != nil {
			return nil, err
		}

		eventID, inserted, err := insertEvent(ctx, tx, userID, event)
		if err != nil {
			return nil, err
		}
		if err := handleUsageEventInsertOutcome(ctx, tx, userID, response, eventID, inserted, event, r.consumeMatchedMeters); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit usage event ingest: %w", err)
	}

	return response, nil
}

func handleUsageEventInsertOutcome(ctx context.Context, tx pgx.Tx, userID uuid.UUID, response *IngestResponse, eventID *uuid.UUID, inserted bool, event CreateParams, consume usageEventConsumer) error {
	if !inserted {
		response.Duplicates++
		return nil
	}
	if eventID == nil || *eventID == uuid.Nil {
		return errors.New("inserted usage event is missing id")
	}

	response.Inserted++
	return consume(ctx, tx, userID, *eventID, event)
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*UsageEvent, error) {
	const query = `
SELECT
    e.id,
    e.user_id,
    e.parent_id,
    e.timestamp,
    e.name,
    e.source,
    e.customer_id,
    e.external_customer_id,
    e.external_id,
    e.metadata,
    e.created_at,
    (
        SELECT COUNT(*)
        FROM usage_events child
        WHERE child.parent_id = e.id
    ) AS child_count
FROM usage_events e
WHERE e.user_id = $1 AND e.id = $2`

	event, err := scanEvent(r.db.QueryRow(ctx, query, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get usage event: %w", err)
	}

	return event, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	params = normalizeListParams(params)
	where, args := buildListWhere(params)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM usage_events e WHERE %s`, where)
	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count usage events: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	queryArgs := append(args, params.Limit, offset)
	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	orderBy := "e.timestamp DESC"
	if params.Sorting == "timestamp" {
		orderBy = "e.timestamp ASC"
	}

	query := fmt.Sprintf(`
SELECT
    e.id,
    e.user_id,
    e.parent_id,
    e.timestamp,
    e.name,
    e.source,
    e.customer_id,
    e.external_customer_id,
    e.external_id,
    e.metadata,
    e.created_at,
    (
        SELECT COUNT(*)
        FROM usage_events child
        WHERE child.parent_id = e.id
    ) AS child_count
FROM usage_events e
WHERE %s
ORDER BY %s
LIMIT $%d OFFSET $%d`, where, orderBy, limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list usage events: %w", err)
	}
	defer rows.Close()

	events := make([]UsageEvent, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan usage event: %w", err)
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage events: %w", err)
	}

	maxPage := 0
	if totalCount > 0 {
		maxPage = (totalCount + params.Limit - 1) / params.Limit
	}

	return &ListResponse{
		Items: events,
		Pagination: Pagination{
			TotalCount: totalCount,
			Page:       params.Page,
			Limit:      params.Limit,
			MaxPage:    maxPage,
		},
	}, nil
}

func (r *Repository) resolveEventCustomer(ctx context.Context, tx pgx.Tx, userID uuid.UUID, event *CreateParams) error {
	if event.CustomerID != nil || strings.TrimSpace(event.ExternalCustomerID) == "" {
		return nil
	}

	const query = `SELECT id FROM customers WHERE user_id = $1 AND external_id = $2`
	var customerID uuid.UUID
	err := tx.QueryRow(ctx, query, userID, strings.TrimSpace(event.ExternalCustomerID)).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve event customer: %w", err)
	}

	event.CustomerID = &customerID
	return nil
}

func (r *Repository) validateReferences(ctx context.Context, tx pgx.Tx, userID uuid.UUID, event CreateParams) error {
	if event.ParentID != nil {
		exists, err := existsForUser(ctx, tx, "usage_events", userID, *event.ParentID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: parent event not found", ErrNotFound)
		}
	}

	if event.CustomerID != nil {
		exists, err := existsForUser(ctx, tx, "customers", userID, *event.CustomerID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: customer not found", ErrNotFound)
		}
	}

	return nil
}

func insertEvent(ctx context.Context, tx pgx.Tx, userID uuid.UUID, event CreateParams) (*uuid.UUID, bool, error) {
	const query = `
INSERT INTO usage_events (
    user_id,
    parent_id,
    timestamp,
    name,
    source,
    customer_id,
    external_customer_id,
    external_id,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9)
ON CONFLICT (user_id, external_id) WHERE external_id IS NOT NULL DO NOTHING
RETURNING id`

	metadata, err := encodeJSON(defaultMetadata(event.Metadata))
	if err != nil {
		return nil, false, err
	}

	var eventID uuid.UUID
	err = tx.QueryRow(
		ctx,
		query,
		userID,
		event.ParentID,
		event.Timestamp,
		event.Name,
		event.Source,
		event.CustomerID,
		event.ExternalCustomerID,
		event.ExternalID,
		metadata,
	).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("insert usage event: %w", err)
	}

	return &eventID, true, nil
}

func existsForUser(ctx context.Context, tx pgx.Tx, table string, userID, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE user_id = $1 AND id = $2)`, table)

	var exists bool
	if err := tx.QueryRow(ctx, query, userID, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check %s ownership: %w", table, err)
	}

	return exists, nil
}

func buildListWhere(params ListParams) (string, []any) {
	where := []string{"e.user_id = $1"}
	args := []any{params.UserID}

	appendFilter := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if params.Name != "" {
		appendFilter("e.name = $%d", params.Name)
	}
	if params.Source != "" {
		appendFilter("e.source = $%d", params.Source)
	}
	if params.CustomerID != nil {
		appendFilter("e.customer_id = $%d", *params.CustomerID)
	}
	if params.ExternalCustomerID != "" {
		appendFilter("e.external_customer_id = $%d", params.ExternalCustomerID)
	}
	if params.ParentID != nil {
		appendFilter("e.parent_id = $%d", *params.ParentID)
	}
	if params.StartTimestamp != nil {
		appendFilter("e.timestamp >= $%d", *params.StartTimestamp)
	}
	if params.EndTimestamp != nil {
		appendFilter("e.timestamp <= $%d", *params.EndTimestamp)
	}

	return strings.Join(where, " AND "), args
}

func normalizeListParams(params ListParams) ListParams {
	params.Name = strings.TrimSpace(params.Name)
	params.Source = Source(strings.ToLower(strings.TrimSpace(string(params.Source))))
	params.ExternalCustomerID = strings.TrimSpace(params.ExternalCustomerID)
	params.Sorting = strings.TrimSpace(params.Sorting)
	if params.Sorting != "timestamp" {
		params.Sorting = "-timestamp"
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	return params
}

func scanEvent(row pgx.Row) (*UsageEvent, error) {
	var event UsageEvent
	var metadataBytes []byte
	var source string

	if err := row.Scan(
		&event.ID,
		&event.UserID,
		&event.ParentID,
		&event.Timestamp,
		&event.Name,
		&source,
		&event.CustomerID,
		&event.ExternalCustomerID,
		&event.ExternalID,
		&metadataBytes,
		&event.CreatedAt,
		&event.ChildCount,
	); err != nil {
		return nil, err
	}

	event.Source = Source(source)
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &event.Metadata); err != nil {
			return nil, fmt.Errorf("decode usage event metadata: %w", err)
		}
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}

	return &event, nil
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

func nowUTC() time.Time {
	return time.Now().UTC()
}
