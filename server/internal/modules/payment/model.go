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
	ID                uuid.UUID
	UserID            uuid.UUID
	CheckoutID        *uuid.UUID
	CustomerID        *uuid.UUID
	ExternalID        *string
	Provider          string
	ProviderReference *string
	Status            Status
	Currency          string
	Amount            int64
	FeeAmount         int64
	NetAmount         int64
	Metadata          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
