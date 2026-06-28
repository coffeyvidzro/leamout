package meter

import (
	"time"

	"github.com/google/uuid"
)

type Unit string

type AggregationFunc string

const (
	UnitScalar Unit = "scalar"
	UnitToken  Unit = "token"
	UnitCustom Unit = "custom"

	AggregationCount  AggregationFunc = "count"
	AggregationSum    AggregationFunc = "sum"
	AggregationMax    AggregationFunc = "max"
	AggregationMin    AggregationFunc = "min"
	AggregationAvg    AggregationFunc = "avg"
	AggregationUnique AggregationFunc = "unique"
)

type Meter struct {
	ID               uuid.UUID      `json:"id"`
	UserID           uuid.UUID      `json:"user_id"`
	Name             string         `json:"name"`
	EventFilter      EventFilter    `json:"filter"`
	Aggregation      Aggregation    `json:"aggregation"`
	Unit             Unit           `json:"unit"`
	CustomLabel      *string        `json:"custom_label,omitempty"`
	CustomMultiplier *int           `json:"custom_multiplier,omitempty"`
	ArchivedAt       *time.Time     `json:"archived_at,omitempty"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type EventFilter struct {
	Conjunction string         `json:"conjunction" binding:"required,oneof=and or"`
	Clauses     []FilterClause `json:"clauses" binding:"required"`
}

type FilterClause struct {
	Property string `json:"property" binding:"required"`
	Operator string `json:"operator" binding:"required,oneof=eq ne gt gte lt lte contains exists"`
	Value    any    `json:"value,omitempty"`
}

type Aggregation struct {
	Func     AggregationFunc `json:"func" binding:"required,oneof=count sum max min avg unique"`
	Property string          `json:"property,omitempty"`
}

type CreateRequest struct {
	Name             string         `json:"name" binding:"required,min=3,max=160"`
	EventFilter      EventFilter    `json:"filter" binding:"required"`
	Aggregation      Aggregation    `json:"aggregation" binding:"required"`
	Unit             Unit           `json:"unit" binding:"omitempty,oneof=scalar token custom"`
	CustomLabel      *string        `json:"custom_label,omitempty" binding:"omitempty,max=80"`
	CustomMultiplier *int           `json:"custom_multiplier,omitempty" binding:"omitempty,min=1"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type UpdateRequest struct {
	Name             *string        `json:"name,omitempty" binding:"omitempty,min=3,max=160"`
	EventFilter      *EventFilter   `json:"filter,omitempty"`
	Aggregation      *Aggregation   `json:"aggregation,omitempty"`
	Unit             *Unit          `json:"unit,omitempty" binding:"omitempty,oneof=scalar token custom"`
	CustomLabel      *string        `json:"custom_label,omitempty" binding:"omitempty,max=80"`
	CustomMultiplier *int           `json:"custom_multiplier,omitempty" binding:"omitempty,min=1"`
	Archived         *bool          `json:"archived,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type ListParams struct {
	UserID          uuid.UUID
	IncludeArchived bool
	Page            int
	Limit           int
}

type ListResponse struct {
	Items      []Meter    `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	MaxPage    int `json:"max_page"`
}

type QuantityParams struct {
	UserID         uuid.UUID
	MeterID        uuid.UUID
	StartTimestamp *time.Time
	EndTimestamp   *time.Time
}

type QuantityResponse struct {
	MeterID         uuid.UUID  `json:"meter_id"`
	StartTimestamp  *time.Time `json:"start_timestamp,omitempty"`
	EndTimestamp    *time.Time `json:"end_timestamp,omitempty"`
	Quantity        float64    `json:"quantity"`
	Aggregation     string     `json:"aggregation"`
	AggregationUnit string     `json:"unit"`
}
