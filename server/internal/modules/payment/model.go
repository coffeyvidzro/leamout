package payment

import (
	"time"

	"github.com/google/uuid"
)

type Status string

type AttemptStatus string

const (
	StatusPending    Status = "pending"
	StatusAuthorized Status = "authorized"
	StatusCaptured   Status = "captured"
	StatusFailed     Status = "failed"
	StatusRefunded   Status = "refunded"
	StatusVoided     Status = "voided"

	AttemptStatusPending    AttemptStatus = "pending"
	AttemptStatusProcessing AttemptStatus = "processing"
	AttemptStatusSucceeded  AttemptStatus = "succeeded"
	AttemptStatusFailed     AttemptStatus = "failed"
	AttemptStatusCanceled   AttemptStatus = "canceled"
	AttemptStatusExpired    AttemptStatus = "expired"
	AttemptStatusUnknown    AttemptStatus = "unknown"
)

type Payment struct {
	ID                uuid.UUID      `json:"id"`
	UserID            uuid.UUID      `json:"user_id"`
	CheckoutID        *uuid.UUID     `json:"checkout_id,omitempty"`
	CustomerID        *uuid.UUID     `json:"customer_id,omitempty"`
	ExternalID        string         `json:"external_id"`
	Provider          string         `json:"provider"`
	ProviderReference *string        `json:"provider_reference,omitempty"`
	Status            Status         `json:"status"`
	Currency          string         `json:"currency"`
	Amount            int64          `json:"amount"`
	FeeAmount         int64          `json:"fee_amount"`
	NetAmount         int64          `json:"net_amount"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type Attempt struct {
	ID                uuid.UUID      `json:"id"`
	PaymentID         uuid.UUID      `json:"payment_id"`
	Provider          string         `json:"provider"`
	ProviderReference *string        `json:"provider_reference,omitempty"`
	Status            AttemptStatus  `json:"status"`
	ErrorCode         *string        `json:"error_code,omitempty"`
	ErrorMessage      *string        `json:"error_message,omitempty"`
	RawRequest        map[string]any `json:"raw_request"`
	RawResponse       map[string]any `json:"raw_response"`
	AttemptedAt       time.Time      `json:"attempted_at"`
}

type CreateParams struct {
	UserID            uuid.UUID
	CheckoutID        *uuid.UUID
	CustomerID        *uuid.UUID
	ExternalID        string
	Provider          string
	ProviderReference string
	Status            Status
	Currency          string
	Amount            int64
	FeeAmount         int64
	Metadata          map[string]any
}

type UpdateFromProviderParams struct {
	ExternalID        string
	Provider          string
	ProviderReference string
	Status            Status
	Metadata          map[string]any
}

type CreateAttemptParams struct {
	PaymentID         uuid.UUID
	Provider          string
	ProviderReference string
	Status            AttemptStatus
	ErrorCode         string
	ErrorMessage      string
	RawRequest        map[string]any
	RawResponse       map[string]any
}

type ListParams struct {
	UserID uuid.UUID
	Status Status
	Limit  int
	Offset int
}
