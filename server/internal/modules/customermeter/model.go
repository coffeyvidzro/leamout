package customermeter

import (
	"time"

	"github.com/google/uuid"
)

type CustomerMeter struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	MeterID       uuid.UUID       `json:"meter_id"`
	ConsumedUnits float64         `json:"consumed_units"`
	CreditedUnits float64         `json:"credited_units"`
	Balance       float64         `json:"balance"`
	Customer      CustomerSummary `json:"customer"`
	Meter         MeterSummary    `json:"meter"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CustomerSummary struct {
	ID         uuid.UUID      `json:"id"`
	Name       string         `json:"name"`
	Email      *string        `json:"email,omitempty"`
	Phone      string         `json:"phone"`
	ExternalID *string        `json:"external_id,omitempty"`
	Address    map[string]any `json:"address"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type MeterSummary struct {
	ID               uuid.UUID      `json:"id"`
	Name             string         `json:"name"`
	Filter           map[string]any `json:"filter"`
	Aggregation      map[string]any `json:"aggregation"`
	Unit             string         `json:"unit"`
	CustomLabel      *string        `json:"custom_label,omitempty"`
	CustomMultiplier *int           `json:"custom_multiplier,omitempty"`
	ArchivedAt       *time.Time     `json:"archived_at,omitempty"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type ListParams struct {
	UserID             uuid.UUID
	CustomerID         *uuid.UUID
	ExternalCustomerID string
	MeterID            *uuid.UUID
	Page               int
	Limit              int
}

type ListResponse struct {
	Items      []CustomerMeter `json:"items"`
	Pagination Pagination      `json:"pagination"`
}

type Pagination struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	MaxPage    int `json:"max_page"`
}
