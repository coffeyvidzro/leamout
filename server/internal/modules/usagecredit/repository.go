package usagecredit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	if err := r.upsertSubscriptionCreditGrants(ctx, tx, userID, subscriptionID, checkoutID, fallbackCustomerID); err != nil {
		return err
	}
	if err := r.writeGrantLedgerEntries(ctx, tx, userID, subscriptionID, checkoutID); err != nil {
		return err
	}
	if err := r.refreshCustomerMetersForCheckout(ctx, tx, userID, subscriptionID, checkoutID); err != nil {
		return err
	}

	return nil
}

func (r *Repository) ListGrants(ctx context.Context, params ListGrantsParams) (*ListGrantsResponse, error) {
	params = normalizeGrantParams(params)
	where, args := grantWhereClause(params)

	var totalCount int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM meter_credit_grants `+where, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count usage credit grants: %w", err)
	}

	args = append(args, params.Limit, (params.Page-1)*params.Limit)
	query := grantSelectQuery() + where + `
ORDER BY created_at DESC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list usage credit grants: %w", err)
	}
	defer rows.Close()

	items := make([]CreditGrant, 0)
	for rows.Next() {
		grant, err := scanCreditGrant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan usage credit grant: %w", err)
		}
		items = append(items, *grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage credit grants: %w", err)
	}

	return &ListGrantsResponse{
		Items: items,
		Pagination: Pagination{TotalCount: totalCount, Page: params.Page, Limit: params.Limit, MaxPage: maxPage(totalCount, params.Limit)},
	}, nil
}

func (r *Repository) ListLedger(ctx context.Context, params ListLedgerParams) (*ListLedgerResponse, error) {
	params = normalizeLedgerParams(params)
	where, args := ledgerWhereClause(params)

	var totalCount int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM meter_credit_ledger_entries `+where, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count usage credit ledger: %w", err)
	}

	args = append(args, params.Limit, (params.Page-1)*params.Limit)
	query := ledgerSelectQuery() + where + `
ORDER BY created_at DESC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list usage credit ledger: %w", err)
	}
	defer rows.Close()

	items := make([]LedgerEntry, 0)
	for rows.Next() {
		entry, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan usage credit ledger entry: %w", err)
		}
		items = append(items, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage credit ledger: %w", err)
	}

	return &ListLedgerResponse{
		Items: items,
		Pagination: Pagination{TotalCount: totalCount, Page: params.Page, Limit: params.Limit, MaxPage: maxPage(totalCount, params.Limit)},
	}, nil
}

func (r *Repository) upsertSubscriptionCreditGrants(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	const query = `
WITH target_grants AS (
	SELECT
		bg.user_id,
		COALESCE(bg.customer_id, $4) AS customer_id,
		bg.subscription_id,
		bg.id AS benefit_grant_id,
		(bg.properties->>'meter_id')::uuid AS meter_id,
		(bg.properties->>'quantity')::numeric AS quantity,
		bg.starts_at,
		bg.ends_at
	FROM benefit_grants bg
	JOIN benefits b
	  ON b.user_id = bg.user_id
	 AND b.id = bg.benefit_id
	WHERE bg.user_id = $1
	  AND bg.subscription_id = $2
	  AND COALESCE(bg.customer_id, $4) IS NOT NULL
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
)
INSERT INTO meter_credit_grants (
	user_id,
	customer_id,
	meter_id,
	benefit_grant_id,
	subscription_id,
	source_type,
	source_id,
	status,
	quantity,
	remaining_quantity,
	starts_at,
	expires_at,
	metadata
)
SELECT
	user_id,
	customer_id,
	meter_id,
	benefit_grant_id,
	subscription_id,
	'checkout',
	$3,
	'active',
	quantity,
	quantity,
	starts_at,
	ends_at,
	jsonb_build_object(
		'source', 'checkout',
		'checkout_session_id', $3::text,
		'benefit_grant_id', benefit_grant_id::text,
		'subscription_id', subscription_id::text
	)
FROM target_grants
ON CONFLICT (user_id, source_type, source_id, benefit_grant_id, meter_id)
DO UPDATE SET
	customer_id = EXCLUDED.customer_id,
	subscription_id = EXCLUDED.subscription_id,
	status = 'active',
	quantity = EXCLUDED.quantity,
	remaining_quantity = LEAST(
		EXCLUDED.quantity,
		GREATEST(0, EXCLUDED.quantity - (meter_credit_grants.quantity - meter_credit_grants.remaining_quantity))
	),
	starts_at = EXCLUDED.starts_at,
	expires_at = EXCLUDED.expires_at,
	metadata = meter_credit_grants.metadata || EXCLUDED.metadata,
	updated_at = NOW()`

	if _, err := tx.Exec(ctx, query, userID, subscriptionID, checkoutID, fallbackCustomerID); err != nil {
		return fmt.Errorf("upsert usage credit grants: %w", err)
	}

	return nil
}

func (r *Repository) writeGrantLedgerEntries(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID) error {
	const query = `
INSERT INTO meter_credit_ledger_entries (
	user_id,
	grant_id,
	customer_id,
	meter_id,
	direction,
	reason,
	quantity,
	balance_after,
	idempotency_key,
	metadata
)
SELECT
	g.user_id,
	g.id,
	g.customer_id,
	g.meter_id,
	'credit',
	'grant',
	g.quantity,
	g.remaining_quantity,
	'checkout:' || g.source_id::text || ':credit-grant:' || g.id::text,
	jsonb_build_object(
		'source', 'checkout',
		'checkout_session_id', g.source_id::text,
		'benefit_grant_id', g.benefit_grant_id::text,
		'subscription_id', g.subscription_id::text
	)
FROM meter_credit_grants g
WHERE g.user_id = $1
  AND g.subscription_id = $2
  AND g.source_type = 'checkout'
  AND g.source_id = $3
ON CONFLICT (user_id, idempotency_key)
WHERE idempotency_key IS NOT NULL
DO NOTHING`

	if _, err := tx.Exec(ctx, query, userID, subscriptionID, checkoutID); err != nil {
		return fmt.Errorf("write usage credit ledger entries: %w", err)
	}

	return nil
}

func (r *Repository) refreshCustomerMetersForCheckout(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID) error {
	const query = `
WITH affected AS (
	SELECT DISTINCT user_id, customer_id, meter_id
	FROM meter_credit_grants
	WHERE user_id = $1
	  AND subscription_id = $2
	  AND source_type = 'checkout'
	  AND source_id = $3
), totals AS (
	SELECT
		g.user_id,
		g.customer_id,
		g.meter_id,
		SUM(g.quantity) AS credited_units,
		SUM(g.quantity - g.remaining_quantity) AS consumed_units,
		SUM(g.remaining_quantity) AS balance
	FROM meter_credit_grants g
	JOIN affected a
	  ON a.user_id = g.user_id
	 AND a.customer_id = g.customer_id
	 AND a.meter_id = g.meter_id
	WHERE g.status = 'active'
	  AND (g.starts_at IS NULL OR g.starts_at <= NOW())
	  AND (g.expires_at IS NULL OR g.expires_at > NOW())
	GROUP BY g.user_id, g.customer_id, g.meter_id
)
INSERT INTO customer_meters (user_id, customer_id, meter_id, consumed_units, credited_units, balance)
SELECT user_id, customer_id, meter_id, consumed_units, credited_units, balance
FROM totals
ON CONFLICT (user_id, customer_id, meter_id)
DO UPDATE SET
	consumed_units = EXCLUDED.consumed_units,
	credited_units = EXCLUDED.credited_units,
	balance = EXCLUDED.balance,
	updated_at = NOW()`

	if _, err := tx.Exec(ctx, query, userID, subscriptionID, checkoutID); err != nil {
		return fmt.Errorf("refresh customer meters from usage credits: %w", err)
	}

	return nil
}

func grantSelectQuery() string {
	return `
SELECT id, user_id, customer_id, meter_id, benefit_grant_id, subscription_id, source_type, source_id,
	status, quantity::float8, remaining_quantity::float8, starts_at, expires_at, rollover_enabled,
	metadata, created_at, updated_at
FROM meter_credit_grants `
}

func ledgerSelectQuery() string {
	return `
SELECT id, user_id, grant_id, customer_id, meter_id, usage_event_id, direction, reason,
	quantity::float8, balance_after::float8, idempotency_key, metadata, created_at
FROM meter_credit_ledger_entries `
}

func grantWhereClause(params ListGrantsParams) (string, []any) {
	args := []any{params.UserID}
	clauses := []string{"user_id = $1"}
	if params.CustomerID != nil {
		args = append(args, *params.CustomerID)
		clauses = append(clauses, fmt.Sprintf("customer_id = $%d", len(args)))
	}
	if params.MeterID != nil {
		args = append(args, *params.MeterID)
		clauses = append(clauses, fmt.Sprintf("meter_id = $%d", len(args)))
	}
	if params.SubscriptionID != nil {
		args = append(args, *params.SubscriptionID)
		clauses = append(clauses, fmt.Sprintf("subscription_id = $%d", len(args)))
	}
	if strings.TrimSpace(params.Status) != "" {
		args = append(args, strings.TrimSpace(params.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func ledgerWhereClause(params ListLedgerParams) (string, []any) {
	args := []any{params.UserID}
	clauses := []string{"user_id = $1"}
	if params.CustomerID != nil {
		args = append(args, *params.CustomerID)
		clauses = append(clauses, fmt.Sprintf("customer_id = $%d", len(args)))
	}
	if params.MeterID != nil {
		args = append(args, *params.MeterID)
		clauses = append(clauses, fmt.Sprintf("meter_id = $%d", len(args)))
	}
	if params.GrantID != nil {
		args = append(args, *params.GrantID)
		clauses = append(clauses, fmt.Sprintf("grant_id = $%d", len(args)))
	}
	if strings.TrimSpace(params.Direction) != "" {
		args = append(args, strings.TrimSpace(params.Direction))
		clauses = append(clauses, fmt.Sprintf("direction = $%d", len(args)))
	}
	if strings.TrimSpace(params.Reason) != "" {
		args = append(args, strings.TrimSpace(params.Reason))
		clauses = append(clauses, fmt.Sprintf("reason = $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanCreditGrant(row pgx.Row) (*CreditGrant, error) {
	var grant CreditGrant
	var subscriptionID pgtype.UUID
	var metadataBytes []byte

	if err := row.Scan(
		&grant.ID,
		&grant.UserID,
		&grant.CustomerID,
		&grant.MeterID,
		&grant.BenefitGrantID,
		&subscriptionID,
		&grant.SourceType,
		&grant.SourceID,
		&grant.Status,
		&grant.Quantity,
		&grant.RemainingQuantity,
		&grant.StartsAt,
		&grant.ExpiresAt,
		&grant.RolloverEnabled,
		&metadataBytes,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	); err != nil {
		return nil, err
	}
	grant.SubscriptionID = uuidFromPg(subscriptionID)
	if err := decodeJSONMap(metadataBytes, &grant.Metadata, "usage credit grant metadata"); err != nil {
		return nil, err
	}

	return &grant, nil
}

func scanLedgerEntry(row pgx.Row) (*LedgerEntry, error) {
	var entry LedgerEntry
	var usageEventID pgtype.UUID
	var idempotencyKey sql.NullString
	var metadataBytes []byte

	if err := row.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.GrantID,
		&entry.CustomerID,
		&entry.MeterID,
		&usageEventID,
		&entry.Direction,
		&entry.Reason,
		&entry.Quantity,
		&entry.BalanceAfter,
		&idempotencyKey,
		&metadataBytes,
		&entry.CreatedAt,
	); err != nil {
		return nil, err
	}
	entry.UsageEventID = uuidFromPg(usageEventID)
	if idempotencyKey.Valid {
		entry.IdempotencyKey = &idempotencyKey.String
	}
	if err := decodeJSONMap(metadataBytes, &entry.Metadata, "usage credit ledger metadata"); err != nil {
		return nil, err
	}

	return &entry, nil
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

func uuidFromPg(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func normalizeGrantParams(params ListGrantsParams) ListGrantsParams {
	params.Page, params.Limit = normalizePage(params.Page, params.Limit)
	params.Status = strings.TrimSpace(params.Status)
	return params
}

func normalizeLedgerParams(params ListLedgerParams) ListLedgerParams {
	params.Page, params.Limit = normalizePage(params.Page, params.Limit)
	params.Direction = strings.TrimSpace(params.Direction)
	params.Reason = strings.TrimSpace(params.Reason)
	return params
}

func normalizePage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func maxPage(totalCount, limit int) int {
	if totalCount == 0 || limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(totalCount) / float64(limit)))
}
