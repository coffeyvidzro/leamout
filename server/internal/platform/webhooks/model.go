package webhooks

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

type DeliveryStatus string

const (
	EventCheckoutCreated      EventType = "checkout.created"
	EventCheckoutCompleted    EventType = "checkout.completed"
	EventPaymentInitiated     EventType = "payment.initiated"
	EventPaymentCaptured      EventType = "payment.captured"
	EventPaymentFailed        EventType = "payment.failed"
	EventSubscriptionCreated  EventType = "subscription.created"
	EventSubscriptionRenewed  EventType = "subscription.renewed"
	EventSubscriptionCanceled EventType = "subscription.canceled"
	EventDunningSent          EventType = "dunning.sent"
	EventDunningClicked       EventType = "dunning.clicked"
	EventDunningPaid          EventType = "dunning.paid"
	EventUsageIngested        EventType = "usage.ingested"
	EventCreditsGranted       EventType = "credits.granted"
	EventCreditsConsumed      EventType = "credits.consumed"
	EventWalletCredited       EventType = "wallet.credited"
	EventWalletDebited        EventType = "wallet.debited"
)

const (
	DeliveryPending    DeliveryStatus = "pending"
	DeliveryProcessing DeliveryStatus = "processing"
	DeliveryDelivered  DeliveryStatus = "delivered"
	DeliveryFailed     DeliveryStatus = "failed"
)

type Endpoint struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	URL         string         `json:"url"`
	Description *string        `json:"description,omitempty"`
	SecretHash  string         `json:"-"`
	Enabled     bool           `json:"enabled"`
	EventTypes  []EventType    `json:"event_types"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Delivery struct {
	ID             uuid.UUID      `json:"id"`
	EndpointID     uuid.UUID      `json:"endpoint_id"`
	UserID         uuid.UUID      `json:"user_id"`
	EventID        uuid.UUID      `json:"event_id"`
	EventType      EventType      `json:"event_type"`
	AggregateType  string         `json:"aggregate_type"`
	AggregateID    uuid.UUID      `json:"aggregate_id"`
	IdempotencyKey *string        `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
	Status         DeliveryStatus `json:"status"`
	Attempts       int            `json:"attempts"`
	NextAttemptAt  time.Time      `json:"next_attempt_at"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	LastStatusCode *int           `json:"last_status_code,omitempty"`
	LastError      *string        `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type EnqueueParams struct {
	UserID         uuid.UUID
	EventType      EventType
	AggregateType  string
	AggregateID    uuid.UUID
	IdempotencyKey string
	Payload        map[string]any
	NextAttemptAt  time.Time
}
