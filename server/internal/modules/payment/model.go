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

type Attempt struct {
	ID                uuid.UUID
	PaymentID         uuid.UUID
	Provider          string
	ProviderReference *string
	Status            AttemptStatus
	ErrorCode         *string
	ErrorMessage      *string
	RawRequest        map[string]any
	RawResponse       map[string]any
	AttemptedAt       time.Time
}

type StartCheckoutPaymentParams struct {
	CheckoutID    uuid.UUID
	UserID        uuid.UUID
	CustomerID    *uuid.UUID
	Amount        int64
	FeeAmount     int64
	Currency      string
	Country       string
	Phone         string
	Operator      string
	CustomerName  string
	CustomerEmail string
	Label         string
	ReturnURL     string
	Metadata      map[string]string
}

type StartCheckoutPaymentResult struct {
	PaymentID         uuid.UUID
	CheckoutSessionID uuid.UUID
	ExternalRef       string
	ProviderID        string
	ProviderReference string
	Status            Status
	AttemptStatus     AttemptStatus
	NextActionType    string
	NextActionURL     string
	CustomerMessage   string
}

type CreateParams struct {
	UserID     uuid.UUID
	CheckoutID *uuid.UUID
	CustomerID *uuid.UUID
	ExternalID string
	Provider   string
	Status     Status
	Currency   string
	Amount     int64
	FeeAmount  int64
	Metadata   map[string]any
}

type UpdateFromProviderParams struct {
	ExternalID        string
	Provider          string
	ProviderReference string
	Status            Status
	RawResponse       []byte
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
	RawResponse       []byte
}

type ListParams struct {
	UserID uuid.UUID
	Status Status
	Limit  int
	Offset int
}
