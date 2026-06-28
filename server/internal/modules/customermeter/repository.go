package customermeter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("customer meter not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*CustomerMeter, error) {
	query := baseSelectQuery() + `
WHERE cm.user_id = $1
  AND cm.id = $2`

	meter, err := scanCustomerMeter(r.db.QueryRow(ctx, query, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get customer meter: %w", err)
	}

	return meter, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	params = normalizeListParams(params)

	where, args := listWhereClause(params)
	countQuery := `SELECT COUNT(*) FROM customer_meters cm JOIN customers c ON c.user_id = cm.user_id AND c.id = cm.customer_id ` + where

	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count customer meters: %w", err)
	}

	args = append(args, params.Limit, (params.Page-1)*params.Limit)
	query := baseSelectQuery() + where + `
ORDER BY cm.created_at DESC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list customer meters: %w", err)
	}
	defer rows.Close()

	items := make([]CustomerMeter, 0)
	for rows.Next() {
		meter, err := scanCustomerMeter(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer meter: %w", err)
		}
		items = append(items, *meter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer meters: %w", err)
	}

	return &ListResponse{
		Items: items,
		Pagination: Pagination{
			TotalCount: totalCount,
			Page:       params.Page,
			Limit:      params.Limit,
			MaxPage:    maxPage(totalCount, params.Limit),
		},
	}, nil
}

func (r *Repository) RefreshCreditsForSubscription(ctx context.Context, tx pgx.Tx, userID, subscriptionID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	const query = `
WITH target_customer AS (
	SELECT COALESCE(s.customer_id, $3) AS customer_id
	FROM subscriptions s
	WHERE s.user_id = $1
	  AND s.id = $2
), credit_totals AS (
	SELECT
		bg.user_id,
		bg.customer_id,
		(bg.properties->>'meter_id')::uuid AS meter_id,
		SUM((bg.properties->>'quantity')::numeric) AS credited_units
	FROM benefit_grants bg
	JOIN benefits b
	  ON b.user_id = bg.user_id
	 AND b.id = bg.benefit_id
	JOIN target_customer tc
	  ON tc.customer_id = bg.customer_id
	WHERE bg.user_id = $1
	  AND tc.customer_id IS NOT NULL
	  AND bg.status = 'active'
	  AND bg.revoked_at IS NULL
	  AND (bg.starts_at IS NULL OR bg.starts_at <= NOW())
	  AND (bg.ends_at IS NULL OR bg.ends_at > NOW())
	  AND b.type = 'meter_credit'
	  AND b.archived_at IS NULL
	  AND bg.properties ? 'meter_id'
	  AND bg.properties ? 'quantity'
	  AND (bg.properties->>'meter_id') ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
	  AND (bg.properties->>'quantity') ~ '^[0-9]+(\.[0-9]+)?$'
	GROUP BY bg.user_id, bg.customer_id, (bg.properties->>'meter_id')::uuid
)
INSERT INTO customer_meters (user_id, customer_id, meter_id, credited_units, balance)
SELECT user_id, customer_id, meter_id, credited_units, credited_units
FROM credit_totals
ON CONFLICT (user_id, customer_id, meter_id)
DO UPDATE SET
	credited_units = EXCLUDED.credited_units,
	balance = EXCLUDED.credited_units - customer_meters.consumed_units,
	updated_at = NOW()`

	if _, err := tx.Exec(ctx, query, userID, subscriptionID, fallbackCustomerID); err != nil {
		return fmt.Errorf("refresh customer meter credits: %w", err)
	}

	return nil
}

func baseSelectQuery() string {
	return `
SELECT
	cm.id,
	cm.user_id,
	cm.customer_id,
	cm.meter_id,
	cm.consumed_units::float8,
	cm.credited_units::float8,
	cm.balance::float8,
	cm.created_at,
	cm.updated_at,
	c.id,
	c.name,
	c.email,
	c.phone,
	c.external_id,
	c.address,
	c.metadata,
	c.created_at,
	c.updated_at,
	m.id,
	m.name,
	m.event_filter,
	m.aggregation,
	m.unit,
	m.custom_label,
	m.custom_multiplier,
	m.archived_at,
	m.metadata,
	m.created_at,
	m.updated_at
FROM customer_meters cm
JOIN customers c
  ON c.user_id = cm.user_id
 AND c.id = cm.customer_id
JOIN meters m
  ON m.user_id = cm.user_id
 AND m.id = cm.meter_id
`
}

func listWhereClause(params ListParams) (string, []any) {
	args := []any{params.UserID}
	clauses := []string{"cm.user_id = $1"}

	if params.CustomerID != nil {
		args = append(args, *params.CustomerID)
		clauses = append(clauses, fmt.Sprintf("cm.customer_id = $%d", len(args)))
	}
	if strings.TrimSpace(params.ExternalCustomerID) != "" {
		args = append(args, strings.TrimSpace(params.ExternalCustomerID))
		clauses = append(clauses, fmt.Sprintf("c.external_id = $%d", len(args)))
	}
	if params.MeterID != nil {
		args = append(args, *params.MeterID)
		clauses = append(clauses, fmt.Sprintf("cm.meter_id = $%d", len(args)))
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanCustomerMeter(row pgx.Row) (*CustomerMeter, error) {
	var meter CustomerMeter
	var customerAddressBytes []byte
	var customerMetadataBytes []byte
	var meterFilterBytes []byte
	var meterAggregationBytes []byte
	var meterMetadataBytes []byte

	if err := row.Scan(
		&meter.ID,
		&meter.UserID,
		&meter.CustomerID,
		&meter.MeterID,
		&meter.ConsumedUnits,
		&meter.CreditedUnits,
		&meter.Balance,
		&meter.CreatedAt,
		&meter.UpdatedAt,
		&meter.Customer.ID,
		&meter.Customer.Name,
		&meter.Customer.Email,
		&meter.Customer.Phone,
		&meter.Customer.ExternalID,
		&customerAddressBytes,
		&customerMetadataBytes,
		&meter.Customer.CreatedAt,
		&meter.Customer.UpdatedAt,
		&meter.Meter.ID,
		&meter.Meter.Name,
		&meterFilterBytes,
		&meterAggregationBytes,
		&meter.Meter.Unit,
		&meter.Meter.CustomLabel,
		&meter.Meter.CustomMultiplier,
		&meter.Meter.ArchivedAt,
		&meterMetadataBytes,
		&meter.Meter.CreatedAt,
		&meter.Meter.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if err := decodeJSONMap(customerAddressBytes, &meter.Customer.Address, "customer address"); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(customerMetadataBytes, &meter.Customer.Metadata, "customer metadata"); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(meterFilterBytes, &meter.Meter.Filter, "meter filter"); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(meterAggregationBytes, &meter.Meter.Aggregation, "meter aggregation"); err != nil {
		return nil, err
	}
	if err := decodeJSONMap(meterMetadataBytes, &meter.Meter.Metadata, "meter metadata"); err != nil {
		return nil, err
	}

	return &meter, nil
}

func decodeJSONMap(data []byte, target *map[string]any, label string) error {
	if len(data) > 0 {
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("decode %s: %w", label, err)
		}
	}
	if *target == nil {
		*target = map[string]any{}
	}

	return nil
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

func maxPage(totalCount, limit int) int {
	if totalCount == 0 || limit <= 0 {
		return 0
	}

	return int(math.Ceil(float64(totalCount) / float64(limit)))
}
