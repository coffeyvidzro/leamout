package checkout

import (
	"time"

	"github.com/google/uuid"
)

type Mode string

type Source string

type Status string

type PaymentAttemptStatus string

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

	PaymentAttemptStatusPending    PaymentAttemptStatus = "pending"
	PaymentAttemptStatusProcessing PaymentAttemptStatus = "processing"
	PaymentAttemptStatusSucceeded  PaymentAttemptStatus = "succeeded"
	PaymentAttemptStatusFailed     PaymentAttemptStatus = "failed"
	PaymentAttemptStatusCanceled   PaymentAttemptStatus = "canceled"
	PaymentAttemptStatusExpired    PaymentAttemptStatus = "expired"
	PaymentAttemptStatusUnknown    PaymentAttemptStatus = "unknown"
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

type RequestIntelligence struct {
	DetectedCountry string `json:"-"`
	DetectedSource  string `json:"-"`
	ClientIP        string `json:"-"`
}

type QuoteRequest struct {
	Country      string              `json:"country" binding:"required"`
	Phone        string              `json:"phone" binding:"required"`
	Operator     string              `json:"operator" binding:"required"`
	Intelligence RequestIntelligence `json:"-"`
}

type QuoteResponse struct {
	CheckoutSessionID string `json:"checkout_session_id"`
	Country           string `json:"country"`
	Currency          string `json:"currency"`
	Method            string `json:"method"`
	Operator          string `json:"operator"`
	BaseAmount        int64  `json:"base_amount"`
	ProcessingFee     int64  `json:"processing_fee"`
	PayableAmount     int64  `json:"payable_amount"`
	FeeRateBps        int64  `json:"fee_rate_bps"`
	FeeFixedAmount    int64  `json:"fee_fixed_amount"`
	FeeMode           string `json:"fee_mode"`
	DetectedCountry   string `json:"detected_country,omitempty"`
	CountryMismatch   bool   `json:"country_mismatch,omitempty"`
}

type PayRequest struct {
	Country       string              `json:"country" binding:"required"`
	Phone         string              `json:"phone" binding:"required"`
	Operator      string              `json:"operator" binding:"required"`
	CustomerName  string              `json:"customer_name" binding:"omitempty,max=160"`
	CustomerEmail string              `json:"customer_email" binding:"omitempty,email"`
	Intelligence  RequestIntelligence `json:"-"`
}

type PayResponse struct {
	CheckoutSessionID string         `json:"checkout_session_id"`
	ExternalRef       string         `json:"external_ref"`
	ProviderID        string         `json:"provider_id"`
	ProviderReference string         `json:"provider_reference,omitempty"`
	Status            string         `json:"status"`
	NextActionType    string         `json:"next_action_type"`
	NextActionURL     string         `json:"next_action_url,omitempty"`
	CustomerMessage   string         `json:"customer_message,omitempty"`
	Quote             *QuoteResponse `json:"quote,omitempty"`
}

type CreatePaymentAttemptParams struct {
	CheckoutSessionID uuid.UUID
	UserID            uuid.UUID
	ExternalRef       string
	ProviderID        string
	ProviderReference string
	Status            PaymentAttemptStatus
	Amount            int64
	Currency          string
	Country           string
	PaymentMethod     string
	Operator          string
	CustomerPhone     string
	ProviderResponse  []byte
	Metadata          map[string]string
}

type ApplyPaymentResultParams struct {
	ExternalRef       string
	ProviderID        string
	ProviderReference string
	Status            PaymentAttemptStatus
	ProviderResponse  []byte
	Metadata          map[string]string
}
