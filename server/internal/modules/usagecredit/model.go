package usagecredit

import (
	"time"

	"github.com/google/uuid"
)

type GrantStatus string

type LedgerDirection string

type LedgerReason string

const (
	GrantStatusActive   GrantStatus = "active"
	GrantStatusDepleted GrantStatus = "depleted"
	GrantStatusExpired  GrantStatus = "expired"
	GrantStatusVoided   GrantStatus = "voided"

	LedgerDirectionCredit LedgerDirection = "credit"
	LedgerDirectionDebit  LedgerDirection = "debit"

	LedgerReasonGrant   LedgerReason = "grant"
	LedgerReasonConsume LedgerReason = "consume"
	LedgerReasonExpire  LedgerReason = "expire"
	LedgerReasonAdjust  LedgerReason = "adjust"
	LedgerReasonRefund  LedgerReason = "refund"
)

type CreditGrant struct {
	ID                uuid.UUID      `json:"id"`
	UserID            uuid.UUID      `json:"user_id"`
	CustomerID        uuid.UUID      `json:"customer_id"`
	MeterID           uuid.UUID      `json:"meter_id"`
	BenefitGrantID    uuid.UUID      `json:"benefit_grant_id"`
	SubscriptionID    *uuid.UUID     `json:"subscription_id,omitempty"`
	SourceType        string         `json:"source_type"`
	SourceID          uuid.UUID      `json:"source_id"`
	Status            GrantStatus    `json:"status"`
	Quantity          float64        `json:"quantity"`
	RemainingQuantity float64        `json:"remaining_quantity"`
	StartsAt          *time.Time     `json:"starts_at,omitempty"`
	ExpiresAt         *time.Time     `json:"expires_at,omitempty"`
	RolloverEnabled   bool           `json:"rollover_enabled"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type LedgerEntry struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	GrantID        uuid.UUID       `json:"grant_id"`
	CustomerID     uuid.UUID       `json:"customer_id"`
	MeterID        uuid.UUID       `json:"meter_id"`
	UsageEventID   *uuid.UUID      `json:"usage_event_id,omitempty"`
	Direction      LedgerDirection `json:"direction"`
	Reason         LedgerReason    `json:"reason"`
	Quantity       float64         `json:"quantity"`
	BalanceAfter   float64         `json:"balance_after"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]any  `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ListGrantsParams struct {
	UserID         uuid.UUID
	CustomerID     *uuid.UUID
	MeterID        *uuid.UUID
	SubscriptionID *uuid.UUID
	Status         string
	Page           int
	Limit          int
}

type ListLedgerParams struct {
	UserID     uuid.UUID
	CustomerID *uuid.UUID
	MeterID    *uuid.UUID
	GrantID    *uuid.UUID
	Direction  string
	Reason     string
	Page       int
	Limit      int
}

type ListGrantsResponse struct {
	Items      []CreditGrant `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

type ListLedgerResponse struct {
	Items      []LedgerEntry `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

type Pagination struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	MaxPage    int `json:"max_page"`
}
