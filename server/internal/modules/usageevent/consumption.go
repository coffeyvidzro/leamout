package usageevent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	metermodule "github.com/cuffeyvidzro/leamout/internal/modules/meter"
	"github.com/cuffeyvidzro/leamout/internal/modules/usagecredit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type meterCandidate struct {
	ID               uuid.UUID
	Filter           metermodule.EventFilter
	Aggregation      metermodule.Aggregation
	CustomMultiplier *int
}

func (r *Repository) consumeMatchedMeters(ctx context.Context, tx pgx.Tx, userID, eventID uuid.UUID, event CreateParams) error {
	if event.CustomerID == nil || *event.CustomerID == uuid.Nil {
		return nil
	}

	meters, err := r.listConsumableMeters(ctx, tx, userID, *event.CustomerID)
	if err != nil {
		return err
	}
	if len(meters) == 0 {
		return nil
	}

	creditRepo := usagecredit.NewRepository(r.db)
	for _, candidate := range meters {
		if !matchesMeterFilter(userID, eventID, event, candidate.Filter) {
			continue
		}

		quantity, err := meterEventQuantity(userID, eventID, event, candidate)
		if err != nil {
			return err
		}
		if quantity <= 0 {
			continue
		}

		idempotencyKey := fmt.Sprintf("usage_event:%s:meter:%s", eventID, candidate.ID)
		if err := creditRepo.ConsumeUsageEvent(ctx, tx, userID, *event.CustomerID, eventID, candidate.ID, quantity, idempotencyKey); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) listConsumableMeters(ctx context.Context, tx pgx.Tx, userID, customerID uuid.UUID) ([]meterCandidate, error) {
	const query = `
SELECT DISTINCT m.id, m.event_filter, m.aggregation, m.custom_multiplier
FROM meter_credit_grants g
JOIN meters m
  ON m.user_id = g.user_id
 AND m.id = g.meter_id
WHERE g.user_id = $1
  AND g.customer_id = $2
  AND g.status = 'active'
  AND g.remaining_quantity > 0
  AND (g.starts_at IS NULL OR g.starts_at <= NOW())
  AND (g.expires_at IS NULL OR g.expires_at > NOW())
  AND m.archived_at IS NULL`

	rows, err := tx.Query(ctx, query, userID, customerID)
	if err != nil {
		return nil, fmt.Errorf("list consumable meters: %w", err)
	}
	defer rows.Close()

	meters := make([]meterCandidate, 0)
	for rows.Next() {
		var candidate meterCandidate
		var filterBytes []byte
		var aggregationBytes []byte
		if err := rows.Scan(&candidate.ID, &filterBytes, &aggregationBytes, &candidate.CustomMultiplier); err != nil {
			return nil, fmt.Errorf("scan consumable meter: %w", err)
		}
		if err := json.Unmarshal(filterBytes, &candidate.Filter); err != nil {
			return nil, fmt.Errorf("decode consumable meter filter: %w", err)
		}
		if err := json.Unmarshal(aggregationBytes, &candidate.Aggregation); err != nil {
			return nil, fmt.Errorf("decode consumable meter aggregation: %w", err)
		}
		meters = append(meters, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consumable meters: %w", err)
	}

	return meters, nil
}

func matchesMeterFilter(userID, eventID uuid.UUID, event CreateParams, filter metermodule.EventFilter) bool {
	if len(filter.Clauses) == 0 {
		return true
	}

	conjunction := strings.ToLower(strings.TrimSpace(filter.Conjunction))
	if conjunction == "or" {
		for _, clause := range filter.Clauses {
			if matchesFilterClause(userID, eventID, event, clause) {
				return true
			}
		}
		return false
	}

	for _, clause := range filter.Clauses {
		if !matchesFilterClause(userID, eventID, event, clause) {
			return false
		}
	}
	return true
}

func matchesFilterClause(userID, eventID uuid.UUID, event CreateParams, clause metermodule.FilterClause) bool {
	value, exists := eventPropertyValue(userID, eventID, event, clause.Property)
	operator := strings.ToLower(strings.TrimSpace(clause.Operator))

	switch operator {
	case "exists":
		return exists && strings.TrimSpace(valueString(value)) != ""
	case "eq":
		return exists && valueString(value) == valueString(clause.Value)
	case "ne":
		return !exists || valueString(value) != valueString(clause.Value)
	case "contains":
		return exists && strings.Contains(valueString(value), valueString(clause.Value))
	case "gt", "gte", "lt", "lte":
		left, ok := numericValue(value)
		if !exists || !ok {
			return false
		}
		right, ok := numericValue(clause.Value)
		if !ok {
			return false
		}
		return compareNumeric(left, right, operator)
	default:
		return false
	}
}

func meterEventQuantity(userID, eventID uuid.UUID, event CreateParams, candidate meterCandidate) (float64, error) {
	var quantity float64

	switch candidate.Aggregation.Func {
	case metermodule.AggregationCount:
		quantity = 1
	case metermodule.AggregationSum, metermodule.AggregationMax, metermodule.AggregationMin, metermodule.AggregationAvg:
		value, exists := eventPropertyValue(userID, eventID, event, candidate.Aggregation.Property)
		if !exists {
			return 0, nil
		}
		numeric, ok := numericValue(value)
		if !ok {
			return 0, fmt.Errorf("meter %s aggregation property %s is not numeric", candidate.ID, candidate.Aggregation.Property)
		}
		quantity = numeric
	case metermodule.AggregationUnique:
		value, exists := eventPropertyValue(userID, eventID, event, candidate.Aggregation.Property)
		if exists && strings.TrimSpace(valueString(value)) != "" {
			quantity = 1
		}
	default:
		return 0, fmt.Errorf("unsupported meter aggregation %s", candidate.Aggregation.Func)
	}

	if candidate.CustomMultiplier != nil && *candidate.CustomMultiplier > 0 {
		quantity *= float64(*candidate.CustomMultiplier)
	}

	return quantity, nil
}

func eventPropertyValue(userID, eventID uuid.UUID, event CreateParams, property string) (any, bool) {
	property = strings.TrimSpace(property)
	switch property {
	case "id":
		return eventID.String(), true
	case "user_id":
		return userID.String(), true
	case "name":
		return event.Name, true
	case "source":
		return string(event.Source), true
	case "customer_id":
		if event.CustomerID == nil {
			return nil, false
		}
		return event.CustomerID.String(), true
	case "parent_id":
		if event.ParentID == nil {
			return nil, false
		}
		return event.ParentID.String(), true
	case "external_customer_id":
		if event.ExternalCustomerID == "" {
			return nil, false
		}
		return event.ExternalCustomerID, true
	case "external_id":
		if event.ExternalID == "" {
			return nil, false
		}
		return event.ExternalID, true
	case "timestamp":
		return event.Timestamp, true
	}

	metadataPath := strings.TrimPrefix(property, "metadata.")
	parts := strings.Split(metadataPath, ".")
	var current any = event.Metadata
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	case time.Time:
		return float64(typed.Unix()), true
	default:
		return 0, false
	}
}

func compareNumeric(left, right float64, operator string) bool {
	switch operator {
	case "gt":
		return left > right
	case "gte":
		return left >= right
	case "lt":
		return left < right
	case "lte":
		return left <= right
	default:
		return false
	}
}

func valueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
