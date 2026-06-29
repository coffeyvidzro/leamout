package events

import (
	"time"

	"github.com/google/uuid"
)

type Status string

type Name string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
)

const (
	CheckoutCreated     Name = "checkout.created"
	CheckoutCompleted   Name = "checkout.completed"
	PaymentInitiated    Name = "payment.initiated"
	PaymentCaptured     Name = "payment.captured"
	PaymentFailed       Name = "payment.failed"
	SubscriptionRenewed Name = "subscription.renewed"
	DunningSent         Name = "dunning.sent"
	DunningClicked      Name = "dunning.clicked"
	DunningPaid         Name = "dunning.paid"
	UsageIngested       Name = "usage.ingested"
	CreditsGranted      Name = "credits.granted"
	CreditsConsumed     Name = "credits.consumed"
	WalletCredited      Name = "wallet.credited"
	WalletDebited       Name = "wallet.debited"
)

type Event struct {
	ID             uuid.UUID      `json:"id"`
	UserID         *uuid.UUID     `json:"user_id,omitempty"`
	Name           Name           `json:"name"`
	AggregateType  string         `json:"aggregate_type"`
	AggregateID    uuid.UUID      `json:"aggregate_id"`
	IdempotencyKey *string        `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
	Metadata       map[string]any `json:"metadata"`
	Status         Status         `json:"status"`
	Attempts       int            `json:"attempts"`
	AvailableAt    time.Time      `json:"available_at"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	LastError      *string        `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type PublishParams struct {
	UserID         *uuid.UUID
	Name           Name
	AggregateType  string
	AggregateID    uuid.UUID
	IdempotencyKey string
	Payload        map[string]any
	Metadata       map[string]any
	AvailableAt    time.Time
}
