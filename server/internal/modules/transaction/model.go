package transaction

import (
	"time"

	"github.com/google/uuid"
)

type Type string
type Status string

const (
	TypeAuthorization Type = "authorization"
	TypeCapture       Type = "capture"
	TypeRefund        Type = "refund"
	TypeVoid          Type = "void"
	TypeAdjustment    Type = "adjustment"

	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Transaction struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	PaymentID  *uuid.UUID
	CheckoutID *uuid.UUID
	ExternalID *string
	Type       Type
	Status     Status
	Currency   string
	Amount     int64
	OccurredAt time.Time
	Metadata   map[string]any
	CreatedAt  time.Time
}

type CreateParams struct {
	UserID     uuid.UUID
	PaymentID  *uuid.UUID
	CheckoutID *uuid.UUID
	ExternalID string
	Type       Type
	Status     Status
	Currency   string
	Amount     int64
	OccurredAt time.Time
	Metadata   map[string]any
}

type ListParams struct {
	UserID uuid.UUID
	Type   Type
	Limit  int
	Offset int
}
