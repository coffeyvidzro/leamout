package entitlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrNoActiveGrant    = errors.New("no active grant")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Check(ctx context.Context, params CheckParams) (*GrantCandidate, error) {
	customerID, err := r.resolveCustomerID(ctx, params)
	if err != nil {
		return nil, err
	}

	const query = `
SELECT
	bg.id,
	bg.benefit_id,
	b.type,
	b.code,
	bg.ends_at,
	bg.properties,
	CASE
		WHEN b.type = 'meter_credit'
		 AND bg.properties ? 'meter_id'
		 AND (bg.properties->>'meter_id') ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
		THEN (bg.properties->>'meter_id')::uuid
		ELSE NULL
	END AS meter_id,
	cm.id,
	cm.balance::float8
FROM benefit_grants bg
JOIN benefits b
  ON b.user_id = bg.user_id
 AND b.id = bg.benefit_id
LEFT JOIN customer_meters cm
  ON cm.user_id = bg.user_id
 AND cm.customer_id = bg.customer_id
 AND b.type = 'meter_credit'
 AND bg.properties ? 'meter_id'
 AND (bg.properties->>'meter_id') ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 AND cm.meter_id = (bg.properties->>'meter_id')::uuid
WHERE bg.user_id = $1
  AND bg.customer_id = $2
  AND lower(b.code) = lower($3)
  AND bg.status = 'active'
  AND bg.revoked_at IS NULL
  AND (bg.starts_at IS NULL OR bg.starts_at <= NOW())
  AND (bg.ends_at IS NULL OR bg.ends_at > NOW())
  AND b.archived_at IS NULL
ORDER BY bg.granted_at DESC
LIMIT 1`

	candidate := GrantCandidate{CustomerID: customerID}
	var propertiesBytes []byte
	var meterID pgtype.UUID
	var customerMeterID pgtype.UUID
	var balance sql.NullFloat64

	err = r.db.QueryRow(ctx, query, params.UserID, customerID, params.Code).Scan(
		&candidate.GrantID,
		&candidate.BenefitID,
		&candidate.Type,
		&candidate.Code,
		&candidate.EndsAt,
		&propertiesBytes,
		&meterID,
		&customerMeterID,
		&balance,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveGrant
	}
	if err != nil {
		return nil, fmt.Errorf("check entitlement: %w", err)
	}

	if len(propertiesBytes) > 0 {
		if err := json.Unmarshal(propertiesBytes, &candidate.Properties); err != nil {
			return nil, fmt.Errorf("decode benefit grant properties: %w", err)
		}
	}
	if candidate.Properties == nil {
		candidate.Properties = map[string]any{}
	}
	if meterID.Valid {
		id := uuid.UUID(meterID.Bytes)
		candidate.MeterID = &id
	}
	if customerMeterID.Valid {
		id := uuid.UUID(customerMeterID.Bytes)
		candidate.CustomerMeterID = &id
	}
	if balance.Valid {
		value := balance.Float64
		candidate.Balance = &value
	}

	return &candidate, nil
}

func (r *Repository) resolveCustomerID(ctx context.Context, params CheckParams) (uuid.UUID, error) {
	if params.CustomerID != nil {
		const query = `SELECT id FROM customers WHERE user_id = $1 AND id = $2`
		var id uuid.UUID
		if err := r.db.QueryRow(ctx, query, params.UserID, *params.CustomerID).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrCustomerNotFound
		} else if err != nil {
			return uuid.Nil, fmt.Errorf("resolve customer id: %w", err)
		}
		return id, nil
	}

	const query = `SELECT id FROM customers WHERE user_id = $1 AND external_id = $2`
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, query, params.UserID, strings.TrimSpace(params.ExternalCustomerID)).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrCustomerNotFound
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("resolve external customer id: %w", err)
	}

	return id, nil
}
