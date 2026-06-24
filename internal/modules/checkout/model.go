package checkout

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusOpen      Status = "open"
	StatusCompleted Status = "completed"
	StatusExpired   Status = "expired"
	StatusCanceled  Status = "canceled"
)

type Session struct {
	ID               uuid.UUID      `json:"id"`
	UserID           uuid.UUID      `json:"user_id"`
	CustomerID       *uuid.UUID     `json:"customer_id,omitempty"`
	SubscriptionID   uuid.UUID      `json:"subscription_id"`
	DunningAttemptID uuid.UUID      `json:"dunning_attempt_id"`
	DunningTokenID   uuid.UUID      `json:"dunning_token_id"`
	Status           Status         `json:"status"`
	Amount           int64          `json:"amount"`
	Currency         string         `json:"currency"`
	ExpiresAt        time.Time      `json:"expires_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CanceledAt       *time.Time     `json:"canceled_at,omitempty"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type CreateRequest struct {
	CustomerID       *uuid.UUID     `json:"customer_id"`
	SubscriptionID   uuid.UUID      `json:"subscription_id" binding:"required"`
	DunningAttemptID uuid.UUID      `json:"dunning_attempt_id" binding:"required"`
	DunningTokenID   uuid.UUID      `json:"dunning_token_id" binding:"required"`
	ExpiresAt        time.Time      `json:"expires_at" binding:"required"`
	Metadata         map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Status     *Status        `json:"status,omitempty" binding:"omitempty,oneof=open completed expired canceled"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CanceledAt *time.Time     `json:"canceled_at,omitempty"`
}

type CreateSessionParams struct {
	UserID           uuid.UUID
	CustomerID       *uuid.UUID
	SubscriptionID   uuid.UUID
	DunningAttemptID uuid.UUID
	DunningTokenID   uuid.UUID
	ExpiresAt        time.Time
	Metadata         map[string]any
}
