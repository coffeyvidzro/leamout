package usagecredit

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type grantBalance struct {
	ID        uuid.UUID
	Remaining float64
}

func (r *Repository) ConsumeUsageEvent(ctx context.Context, tx pgx.Tx, userID, customerID, usageEventID, meterID uuid.UUID, quantity float64, idempotencyKey string) error {
	if userID == uuid.Nil || customerID == uuid.Nil || usageEventID == uuid.Nil || meterID == uuid.Nil || quantity <= 0 || idempotencyKey == "" {
		return nil
	}

	alreadyConsumed, err := r.hasLedgerEntry(ctx, tx, userID, idempotencyKey)
	if err != nil {
		return err
	}
	if alreadyConsumed {
		return nil
	}

	grants, err := r.lockConsumableGrants(ctx, tx, userID, customerID, meterID)
	if err != nil {
		return err
	}

	remainingToConsume := quantity
	for _, grant := range grants {
		if remainingToConsume <= 0 {
			break
		}
		debit := math.Min(remainingToConsume, grant.Remaining)
		if debit <= 0 {
			continue
		}

		balanceAfter := grant.Remaining - debit
		if err := r.applyGrantDebit(ctx, tx, userID, grant.ID, customerID, meterID, usageEventID, debit, balanceAfter, idempotencyKey); err != nil {
			return err
		}
		remainingToConsume -= debit
	}

	return r.RefreshCustomerMeter(ctx, tx, userID, customerID, meterID)
}

func (r *Repository) hasLedgerEntry(ctx context.Context, tx pgx.Tx, userID uuid.UUID, idempotencyKey string) (bool, error) {
	const query = `
SELECT EXISTS (
	SELECT 1
	FROM meter_credit_ledger_entries
	WHERE user_id = $1
	  AND (
		idempotency_key = $2
		OR idempotency_key LIKE $3
	  )
)`

	var exists bool
	grantEntryPattern := idempotencyKey + ":grant:%"
	if err := tx.QueryRow(ctx, query, userID, idempotencyKey, grantEntryPattern).Scan(&exists); err != nil {
		return false, fmt.Errorf("check usage credit idempotency: %w", err)
	}
	return exists, nil
}

func (r *Repository) lockConsumableGrants(ctx context.Context, tx pgx.Tx, userID, customerID, meterID uuid.UUID) ([]grantBalance, error) {
	const query = `
SELECT id, remaining_quantity::float8
FROM meter_credit_grants
WHERE user_id = $1
  AND customer_id = $2
  AND meter_id = $3
  AND status = 'active'
  AND remaining_quantity > 0
  AND (starts_at IS NULL OR starts_at <= NOW())
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY expires_at NULLS LAST, created_at
FOR UPDATE`

	rows, err := tx.Query(ctx, query, userID, customerID, meterID)
	if err != nil {
		return nil, fmt.Errorf("lock consumable usage credit grants: %w", err)
	}
	defer rows.Close()

	grants := make([]grantBalance, 0)
	for rows.Next() {
		var grant grantBalance
		if err := rows.Scan(&grant.ID, &grant.Remaining); err != nil {
			return nil, fmt.Errorf("scan consumable usage credit grant: %w", err)
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consumable usage credit grants: %w", err)
	}

	return grants, nil
}

func (r *Repository) applyGrantDebit(ctx context.Context, tx pgx.Tx, userID, grantID, customerID, meterID, usageEventID uuid.UUID, debit, balanceAfter float64, baseIdempotencyKey string) error {
	if balanceAfter < 0 {
		balanceAfter = 0
	}

	const updateGrant = `
UPDATE meter_credit_grants
SET remaining_quantity = $3,
	status = 'active',
	updated_at = NOW()
WHERE user_id = $1
  AND id = $2`

	if _, err := tx.Exec(ctx, updateGrant, userID, grantID, balanceAfter); err != nil {
		return fmt.Errorf("debit usage credit grant: %w", err)
	}

	const insertLedger = `
INSERT INTO meter_credit_ledger_entries (
	user_id,
	grant_id,
	customer_id,
	meter_id,
	usage_event_id,
	direction,
	reason,
	quantity,
	balance_after,
	idempotency_key,
	metadata
)
VALUES ($1, $2, $3, $4, $5, 'debit', 'consume', $6, $7, $8, jsonb_build_object(
	'source', 'usage_event',
	'usage_event_id', $5::text,
	'meter_id', $4::text
))
ON CONFLICT (user_id, idempotency_key)
WHERE idempotency_key IS NOT NULL
DO NOTHING`

	entryKey := fmt.Sprintf("%s:grant:%s", baseIdempotencyKey, grantID)
	if _, err := tx.Exec(ctx, insertLedger, userID, grantID, customerID, meterID, usageEventID, debit, balanceAfter, entryKey); err != nil {
		return fmt.Errorf("write usage credit debit ledger entry: %w", err)
	}

	return nil
}

func (r *Repository) RefreshCustomerMeter(ctx context.Context, tx pgx.Tx, userID, customerID, meterID uuid.UUID) error {
	const query = `
WITH totals AS (
	SELECT
		user_id,
		customer_id,
		meter_id,
		SUM(quantity) AS credited_units,
		SUM(quantity - remaining_quantity) AS consumed_units,
		SUM(remaining_quantity) AS balance
	FROM meter_credit_grants
	WHERE user_id = $1
	  AND customer_id = $2
	  AND meter_id = $3
	  AND status = 'active'
	  AND (starts_at IS NULL OR starts_at <= NOW())
	  AND (expires_at IS NULL OR expires_at > NOW())
	GROUP BY user_id, customer_id, meter_id
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

	if _, err := tx.Exec(ctx, query, userID, customerID, meterID); err != nil {
		return fmt.Errorf("refresh customer meter after usage credit consumption: %w", err)
	}
	return nil
}
