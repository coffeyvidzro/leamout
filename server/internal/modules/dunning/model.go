package dunning

import (
	"time"

	"github.com/google/uuid"
)

type AttemptStatus string

type AttemptReason string

type ReminderJobFailureStatus string

const (
	AttemptStatusPending  AttemptStatus = "pending"
	AttemptStatusSent     AttemptStatus = "sent"
	AttemptStatusPaid     AttemptStatus = "paid"
	AttemptStatusExpired  AttemptStatus = "expired"
	AttemptStatusCanceled AttemptStatus = "canceled"

	AttemptReasonRenewalDue    AttemptReason = "renewal_due"
	AttemptReasonPaymentFailed AttemptReason = "payment_failed"

	ReminderJobFailureStatusRetryScheduled ReminderJobFailureStatus = "retry_scheduled"
	ReminderJobFailureStatusRetryExhausted ReminderJobFailureStatus = "retry_exhausted"
)

type Attempt struct {
	ID             uuid.UUID      `json:"id"`
	UserID         uuid.UUID      `json:"user_id"`
	SubscriptionID uuid.UUID      `json:"subscription_id"`
	CustomerID     *uuid.UUID     `json:"customer_id,omitempty"`
	Status         AttemptStatus  `json:"status"`
	Reason         AttemptReason  `json:"reason"`
	PeriodEnd      time.Time      `json:"period_end"`
	ExpiresAt      time.Time      `json:"expires_at"`
	SentAt         *time.Time     `json:"sent_at,omitempty"`
	ClickedAt      *time.Time     `json:"clicked_at,omitempty"`
	PaidAt         *time.Time     `json:"paid_at,omitempty"`
	CanceledAt     *time.Time     `json:"canceled_at,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Token struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	DunningAttemptID uuid.UUID  `json:"dunning_attempt_id"`
	TokenHash        string     `json:"-"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ConversionMetrics struct {
	Sent            int64 `json:"sent"`
	Clicked         int64 `json:"clicked"`
	CheckoutStarted int64 `json:"checkout_started"`
	Paid            int64 `json:"paid"`
	Failed          int64 `json:"failed"`
	Expired         int64 `json:"expired"`
}

type AttemptTransition struct {
	ID             uuid.UUID      `json:"id"`
	UserID         uuid.UUID      `json:"user_id"`
	AttemptID      uuid.UUID      `json:"dunning_attempt_id"`
	Actor          string         `json:"actor"`
	Reason         string         `json:"reason"`
	PreviousStatus AttemptStatus  `json:"previous_status"`
	NextStatus     AttemptStatus  `json:"next_status"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ReminderJobFailure struct {
	ID               uuid.UUID                `json:"id"`
	UserID           uuid.UUID                `json:"user_id"`
	SubscriptionID   uuid.UUID                `json:"subscription_id"`
	CustomerID       uuid.UUID                `json:"customer_id"`
	AttemptID        *uuid.UUID               `json:"dunning_attempt_id,omitempty"`
	CurrentPeriodEnd time.Time                `json:"current_period_end"`
	FailureNumber    int                      `json:"failure_number"`
	Status           ReminderJobFailureStatus `json:"status"`
	ErrorType        string                   `json:"error_type"`
	ErrorMessage     string                   `json:"error_message"`
	Retryable        bool                     `json:"retryable"`
	Metadata         map[string]any           `json:"metadata"`
	CreatedAt        time.Time                `json:"created_at"`
}

type CreateAttemptParams struct {
	UserID         uuid.UUID
	SubscriptionID uuid.UUID
	CustomerID     *uuid.UUID
	Reason         AttemptReason
	PeriodEnd      time.Time
	ExpiresAt      time.Time
	Metadata       map[string]any
}

type CreateTokenParams struct {
	UserID           uuid.UUID
	DunningAttemptID uuid.UUID
	TokenHash        string
	ExpiresAt        time.Time
}

type RecordReminderJobFailureParams struct {
	UserID           uuid.UUID
	SubscriptionID   uuid.UUID
	CustomerID       uuid.UUID
	AttemptID        *uuid.UUID
	CurrentPeriodEnd time.Time
	Status           ReminderJobFailureStatus
	ErrorType        string
	ErrorMessage     string
	Retryable        bool
	Metadata         map[string]any
}

type TokenWithAttempt struct {
	Token   Token   `json:"token"`
	Attempt Attempt `json:"attempt"`
}

type CheckoutDetails struct {
	Amount   int64
	Currency string
}

type ReminderDetails struct {
	CustomerPhone    string
	CurrentPeriodEnd time.Time
}
