package checkout

import (
	"time"

	"github.com/google/uuid"
)

type Mode string

type Source string

type Status string

const (
	ModePayment      Mode = "payment"
	ModeSubscription Mode = "subscription"
	ModeRenewal      Mode = "renewal"

	SourceAPI          Source = "api"
	SourceCheckoutLink Source = "checkout_link"
	SourceDunning      Source = "dunning"
	SourceManual       Source = "manual"

	StatusOpen      Status = "open"
	StatusCompleted Status = "completed"
	StatusExpired   Status = "expired"
	StatusCanceled  Status = "canceled"
)

type Session struct {
	ID               uuid.UUID      `json:"id"`
	UserID           uuid.UUID      `json:"user_id"`
	CustomerID       *uuid.UUID     `json:"customer_id,omitempty"`
	SubscriptionID   *uuid.UUID     `json:"subscription_id,omitempty"`
	Mode             Mode           `json:"mode"`
	Source           Source         `json:"source"`
	Label            *string        `json:"label,omitempty"`
	Amount           int64          `json:"amount"`
	Currency         string         `json:"currency"`
	ClientSecretHash string         `json:"-"`
	ClientSecret     string         `json:"client_secret,omitempty"`
	SuccessURL       *string        `json:"success_url,omitempty"`
	ReturnURL        *string        `json:"return_url,omitempty"`
	Status           Status         `json:"status"`
	ExpiresAt        time.Time      `json:"expires_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CanceledAt       *time.Time     `json:"canceled_at,omitempty"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type CreateRequest struct {
	CustomerID     *uuid.UUID     `json:"customer_id"`
	SubscriptionID *uuid.UUID     `json:"subscription_id"`
	Mode           Mode           `json:"mode" binding:"omitempty,oneof=payment subscription renewal"`
	Source         Source         `json:"source" binding:"omitempty,oneof=api checkout_link dunning manual"`
	Label          *string        `json:"label" binding:"omitempty,max=240"`
	Amount         int64          `json:"amount" binding:"required,gt=0"`
	Currency       string         `json:"currency" binding:"required,len=3,uppercase"`
	SuccessURL     *string        `json:"success_url" binding:"omitempty,url"`
	ReturnURL      *string        `json:"return_url" binding:"omitempty,url"`
	ExpiresAt      time.Time      `json:"expires_at" binding:"required"`
	Metadata       map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Status     *Status        `json:"status,omitempty" binding:"omitempty,oneof=open completed expired canceled"`
	Label      *string        `json:"label,omitempty" binding:"omitempty,max=240"`
	SuccessURL *string        `json:"success_url,omitempty" binding:"omitempty,url"`
	ReturnURL  *string        `json:"return_url,omitempty" binding:"omitempty,url"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CanceledAt *time.Time     `json:"canceled_at,omitempty"`
}
