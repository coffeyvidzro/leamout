package subscription

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive     Status = "active"
	StatusCanceled   Status = "canceled"
	StatusPastDue    Status = "past_due"
	StatusTrialing   Status = "trialing"
	StatusIncomplete Status = "incomplete"
	StatusPaused     Status = "paused"
)

type Subscription struct {
	ID                          uuid.UUID      `json:"id"`
	UserID                      uuid.UUID      `json:"user_id"`
	CustomerID                  *uuid.UUID     `json:"customer_id,omitempty"`
	PriceID                     uuid.UUID      `json:"price_id"`
	Status                      Status         `json:"status"`
	CurrentPeriodStart          time.Time      `json:"current_period_start"`
	CurrentPeriodEnd            time.Time      `json:"current_period_end"`
	CancelAtPeriodEnd           bool           `json:"cancel_at_period_end"`
	CanceledAt                  *time.Time     `json:"canceled_at,omitempty"`
	EndsAt                      *time.Time     `json:"ends_at,omitempty"`
	EndedAt                     *time.Time     `json:"ended_at,omitempty"`
	CustomerCancellationReason  *string        `json:"customer_cancellation_reason,omitempty"`
	CustomerCancellationComment *string        `json:"customer_cancellation_comment,omitempty"`
	Metadata                    map[string]any `json:"metadata"`
	CreatedAt                   time.Time      `json:"created_at"`
	UpdatedAt                   time.Time      `json:"updated_at"`
}

type DunningCandidate struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	CustomerID       *uuid.UUID `json:"customer_id,omitempty"`
	CurrentPeriodEnd time.Time  `json:"current_period_end"`
}

type CreateRequest struct {
	CustomerID         *uuid.UUID     `json:"customer_id"`
	PriceID            uuid.UUID      `json:"price_id" binding:"required"`
	Status             Status         `json:"status" binding:"omitempty,oneof=active canceled past_due trialing incomplete paused"`
	CurrentPeriodStart time.Time      `json:"current_period_start"`
	CurrentPeriodEnd   time.Time      `json:"current_period_end" binding:"required"`
	Metadata           map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Status                      *Status        `json:"status,omitempty" binding:"omitempty,oneof=active canceled past_due trialing incomplete paused"`
	CurrentPeriodEnd            *time.Time     `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd           *bool          `json:"cancel_at_period_end,omitempty"`
	CanceledAt                  *time.Time     `json:"canceled_at,omitempty"`
	EndsAt                      *time.Time     `json:"ends_at,omitempty"`
	EndedAt                     *time.Time     `json:"ended_at,omitempty"`
	CustomerCancellationReason  *string        `json:"customer_cancellation_reason,omitempty" binding:"omitempty,max=240"`
	CustomerCancellationComment *string        `json:"customer_cancellation_comment,omitempty" binding:"omitempty,max=1000"`
	Metadata                    map[string]any `json:"metadata,omitempty"`
}
