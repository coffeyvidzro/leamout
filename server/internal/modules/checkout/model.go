package checkout

import (
	"time"

	"github.com/google/uuid"
)

type Mode string

type Source string

type Status string

type FeePayer string

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

	FeePayerCustomer FeePayer = "customer"
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
	Currency       string         `json:"currency" binding:"required,len=3"`
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

type PublicCheckoutRequest struct {
	Country string `json:"country,omitempty"`
	Network string `json:"network,omitempty"`
}

type CheckoutFeeBreakdown struct {
	FeePayer FeePayer `json:"fee_payer"`

	MMOFeeBps      int64 `json:"mmo_fee_bps"`
	ProviderFeeBps int64 `json:"provider_fee_bps"`
	TotalFeeBps    int64 `json:"total_fee_bps"`

	BaseAmount    int64 `json:"base_amount"`
	ProcessingFee int64 `json:"processing_fee"`
	PayableAmount int64 `json:"payable_amount"`
	NetAmount     int64 `json:"net_amount"`
}

type PublicCheckoutResponse struct {
	ID        uuid.UUID `json:"id"`
	Mode      Mode      `json:"mode"`
	Source    Source    `json:"source"`
	Label     *string   `json:"label,omitempty"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    Status    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`

	Country *string               `json:"country,omitempty"`
	Network *string               `json:"network,omitempty"`
	Fee     *CheckoutFeeBreakdown `json:"fee,omitempty"`

	SuccessURL *string        `json:"success_url,omitempty"`
	ReturnURL  *string        `json:"return_url,omitempty"`
	Metadata   map[string]any `json:"metadata"`
}

type RequestIntelligence struct {
	DetectedCountry string `json:"-"`
	DetectedSource  string `json:"-"`
	ClientIP        string `json:"-"`
}

type PayRequest struct {
	Country       string              `json:"country" binding:"required"`
	Network       string              `json:"network" binding:"required"`
	Phone         string              `json:"phone" binding:"required"`
	CustomerName  string              `json:"customer_name" binding:"omitempty,max=160"`
	CustomerEmail string              `json:"customer_email" binding:"omitempty,email"`
	Intelligence  RequestIntelligence `json:"-"`
}

type PayResponse struct {
	CheckoutSessionID string `json:"checkout_session_id"`
	TransactionID     string `json:"transaction_id"`
	Provider          string `json:"provider"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Status            string `json:"status"`

	// Amount is the actual amount sent to the payment provider.
	Amount        int64                `json:"amount"`
	BaseAmount    int64                `json:"base_amount"`
	ProcessingFee int64                `json:"processing_fee"`
	PayableAmount int64                `json:"payable_amount"`
	Fee           CheckoutFeeBreakdown `json:"fee"`

	Currency        string `json:"currency"`
	Country         string `json:"country"`
	Network         string `json:"network"`
	Phone           string `json:"phone"`
	CustomerMessage string `json:"customer_message,omitempty"`
}
