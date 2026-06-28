package meter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("meter not found")

var safePropertyPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Meter, error) {
	const query = `
INSERT INTO meters (
    user_id,
    name,
    event_filter,
    aggregation,
    unit,
    custom_label,
    custom_multiplier,
    metadata
)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)
RETURNING id, user_id, name, event_filter, aggregation, unit, custom_label, custom_multiplier, archived_at, metadata, created_at, updated_at`

	filterBytes, err := encodeJSON(req.EventFilter)
	if err != nil {
		return nil, err
	}
	aggregationBytes, err := encodeJSON(req.Aggregation)
	if err != nil {
		return nil, err
	}
	metadataBytes, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	meter, err := scanMeter(r.db.QueryRow(
		ctx,
		query,
		userID,
		strings.TrimSpace(req.Name),
		filterBytes,
		aggregationBytes,
		normalizeUnit(req.Unit),
		optionalString(req.CustomLabel),
		req.CustomMultiplier,
		metadataBytes,
	))
	if err != nil {
		return nil, fmt.Errorf("create meter: %w", err)
	}

	return meter, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	params = normalizeListParams(params)
	where := "user_id = $1"
	args := []any{params.UserID}
	if !params.IncludeArchived {
		where += " AND archived_at IS NULL"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM meters WHERE %s`, where)
	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count meters: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	query := fmt.Sprintf(`
SELECT id, user_id, name, event_filter, aggregation, unit, custom_label, custom_multiplier, archived_at, metadata, created_at, updated_at
FROM meters
WHERE %s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, params.Limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list meters: %w", err)
	}
	defer rows.Close()

	meters := make([]Meter, 0)
	for rows.Next() {
		meter, err := scanMeter(rows)
		if err != nil {
			return nil, fmt.Errorf("scan meter: %w", err)
		}
		meters = append(meters, *meter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meters: %w", err)
	}

	maxPage := 0
	if totalCount > 0 {
		maxPage = (totalCount + params.Limit - 1) / params.Limit
	}

	return &ListResponse{
		Items: meters,
		Pagination: Pagination{
			TotalCount: totalCount,
			Page:       params.Page,
			Limit:      params.Limit,
			MaxPage:    maxPage,
		},
	}, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Meter, error) {
	const query = `
SELECT id, user_id, name, event_filter, aggregation, unit, custom_label, custom_multiplier, archived_at, metadata, created_at, updated_at
FROM meters
WHERE user_id = $1 AND id = $2`

	meter, err := scanMeter(r.db.QueryRow(ctx, query, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get meter: %w", err)
	}

	return meter, nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Meter, error) {
	query, args, err := buildUpdateQuery(userID, id, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	meter, err := scanMeter(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update meter: %w", err)
	}

	return meter, nil
}

func (r *Repository) GetQuantities(ctx context.Context, params QuantityParams) (*QuantityResponse, error) {
	meter, err := r.Get(ctx, params.UserID, params.MeterID)
	if err != nil {
		return nil, err
	}

	where, args, err := buildQuantityWhere(params.UserID, meter.EventFilter)
	if err != nil {
		return nil, err
	}
	if params.StartTimestamp != nil {
		args = append(args, *params.StartTimestamp)
		where = append(where, fmt.Sprintf("e.timestamp >= $%d", len(args)))
	}
	if params.EndTimestamp != nil {
		args = append(args, *params.EndTimestamp)
		where = append(where, fmt.Sprintf("e.timestamp <= $%d", len(args)))
	}

	expression, err := aggregationExpression(meter.Aggregation)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT COALESCE(%s, 0)::double precision FROM usage_events e WHERE %s`, expression, strings.Join(where, " AND "))

	var quantity float64
	if err := r.db.QueryRow(ctx, query, args...).Scan(&quantity); err != nil {
		return nil, fmt.Errorf("get meter quantities: %w", err)
	}

	return &QuantityResponse{
		MeterID:         meter.ID,
		StartTimestamp:  params.StartTimestamp,
		EndTimestamp:    params.EndTimestamp,
		Quantity:        quantity,
		Aggregation:     string(meter.Aggregation.Func),
		AggregationUnit: string(meter.Unit),
	}, nil
}

func buildUpdateQuery(userID, id uuid.UUID, req UpdateRequest) (string, []any, error) {
	args := []any{userID, id}
	updates := make([]string, 0, 8)

	add := func(expr string, value any) {
		args = append(args, value)
		updates = append(updates, fmt.Sprintf(expr, len(args)))
	}

	if req.Name != nil {
		add("name = $%d", strings.TrimSpace(*req.Name))
	}
	if req.EventFilter != nil {
		data, err := encodeJSON(req.EventFilter)
		if err != nil {
			return "", nil, err
		}
		add("event_filter = $%d", data)
	}
	if req.Aggregation != nil {
		data, err := encodeJSON(req.Aggregation)
		if err != nil {
			return "", nil, err
		}
		add("aggregation = $%d", data)
	}
	if req.Unit != nil {
		unit := normalizeUnit(*req.Unit)
		add("unit = $%d", unit)
		if unit != UnitCustom {
			updates = append(updates, "custom_label = NULL", "custom_multiplier = NULL")
		}
	}
	if req.CustomLabel != nil {
		add("custom_label = NULLIF($%d, '')", strings.TrimSpace(*req.CustomLabel))
	}
	if req.CustomMultiplier != nil {
		add("custom_multiplier = $%d", *req.CustomMultiplier)
	}
	if req.Archived != nil {
		if *req.Archived {
			updates = append(updates, "archived_at = COALESCE(archived_at, NOW())")
		} else {
			updates = append(updates, "archived_at = NULL")
		}
	}
	if req.Metadata != nil {
		data, err := encodeJSON(req.Metadata)
		if err != nil {
			return "", nil, err
		}
		add("metadata = $%d", data)
	}

	if len(updates) == 0 {
		return "", args, nil
	}

	query := fmt.Sprintf(`
UPDATE meters
SET %s
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, name, event_filter, aggregation, unit, custom_label, custom_multiplier, archived_at, metadata, created_at, updated_at`, strings.Join(updates, ", "))

	return query, args, nil
}

func buildQuantityWhere(userID uuid.UUID, filter EventFilter) ([]string, []any, error) {
	where := []string{"e.user_id = $1"}
	args := []any{userID}

	clauses := make([]string, 0, len(filter.Clauses))
	for _, clause := range filter.Clauses {
		sqlClause, value, hasValue, err := filterClauseSQL(clause)
		if err != nil {
			return nil, nil, err
		}
		if hasValue {
			args = append(args, value)
			sqlClause = fmt.Sprintf(sqlClause, len(args))
		}
		clauses = append(clauses, sqlClause)
	}

	if len(clauses) > 0 {
		joiner := " AND "
		if strings.ToLower(filter.Conjunction) == "or" {
			joiner = " OR "
		}
		where = append(where, "("+strings.Join(clauses, joiner)+")")
	}

	return where, args, nil
}

func filterClauseSQL(clause FilterClause) (string, any, bool, error) {
	property, err := eventPropertyExpression(clause.Property)
	if err != nil {
		return "", nil, false, err
	}

	switch strings.ToLower(strings.TrimSpace(clause.Operator)) {
	case "eq":
		return property + " = $%d", valueToString(clause.Value), true, nil
	case "ne":
		return property + " <> $%d", valueToString(clause.Value), true, nil
	case "contains":
		return property + " ILIKE '%%' || $%d || '%%'", valueToString(clause.Value), true, nil
	case "exists":
		return property + " IS NOT NULL", nil, false, nil
	case "gt", "gte", "lt", "lte":
		op := map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[strings.ToLower(clause.Operator)]
		return fmt.Sprintf("NULLIF(%s, '')::numeric %s $%%d::numeric", property, op), valueToString(clause.Value), true, nil
	default:
		return "", nil, false, fmt.Errorf("unsupported filter operator %s", clause.Operator)
	}
}

func aggregationExpression(aggregation Aggregation) (string, error) {
	switch aggregation.Func {
	case AggregationCount:
		return "COUNT(*)", nil
	case AggregationSum, AggregationMax, AggregationMin, AggregationAvg:
		property, err := eventPropertyExpression(aggregation.Property)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(NULLIF(%s, '')::numeric)", strings.ToUpper(string(aggregation.Func)), property), nil
	case AggregationUnique:
		property, err := eventPropertyExpression(aggregation.Property)
		if err != nil {
			return "", err
		}
		return "COUNT(DISTINCT " + property + ")", nil
	default:
		return "", fmt.Errorf("unsupported aggregation function %s", aggregation.Func)
	}
}

func eventPropertyExpression(property string) (string, error) {
	property = strings.TrimSpace(property)
	if property == "" || !safePropertyPattern.MatchString(property) {
		return "", fmt.Errorf("invalid event property %q", property)
	}

	switch property {
	case "id", "name", "source", "external_customer_id", "external_id":
		return "e." + property + "::text", nil
	case "customer_id", "parent_id":
		return "e." + property + "::text", nil
	}

	metadataPath := property
	metadataPath = strings.TrimPrefix(metadataPath, "metadata.")
	parts := strings.Split(metadataPath, ".")
	for _, part := range parts {
		if part == "" || !safePropertyPattern.MatchString(part) {
			return "", fmt.Errorf("invalid metadata property %q", property)
		}
	}

	return "e.metadata #>> '{" + strings.Join(parts, ",") + "}'", nil
}

func normalizeListParams(params ListParams) ListParams {
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

func scanMeter(row pgx.Row) (*Meter, error) {
	var meter Meter
	var filterBytes []byte
	var aggregationBytes []byte
	var metadataBytes []byte
	var unit string

	if err := row.Scan(
		&meter.ID,
		&meter.UserID,
		&meter.Name,
		&filterBytes,
		&aggregationBytes,
		&unit,
		&meter.CustomLabel,
		&meter.CustomMultiplier,
		&meter.ArchivedAt,
		&metadataBytes,
		&meter.CreatedAt,
		&meter.UpdatedAt,
	); err != nil {
		return nil, err
	}

	meter.Unit = Unit(unit)
	if len(filterBytes) > 0 {
		if err := json.Unmarshal(filterBytes, &meter.EventFilter); err != nil {
			return nil, fmt.Errorf("decode meter filter: %w", err)
		}
	}
	if len(aggregationBytes) > 0 {
		if err := json.Unmarshal(aggregationBytes, &meter.Aggregation); err != nil {
			return nil, fmt.Errorf("decode meter aggregation: %w", err)
		}
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meter.Metadata); err != nil {
			return nil, fmt.Errorf("decode meter metadata: %w", err)
		}
	}
	if meter.Metadata == nil {
		meter.Metadata = map[string]any{}
	}

	return &meter, nil
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

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

func normalizeUnit(unit Unit) Unit {
	if unit == "" {
		return UnitScalar
	}

	return unit
}

func valueToString(value any) string {
	if value == nil {
		return ""
	}

	return fmt.Sprint(value)
}
