package provider

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type ID string

const (
	ProviderPawaPay ID = "pawapay"

	// ProviderMoolre is kept only so older generic routing code can compile while
	// the MVP runtime registers PawaPay as the only payment aggregator.
	ProviderMoolre ID = "moolre"
)

type PaymentMethod string

const (
	PaymentMethodMobileMoney PaymentMethod = "mobile_money"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusExpired    PaymentStatus = "expired"
	PaymentStatusUnknown    PaymentStatus = "unknown"
)

type NextActionType string

const (
	NextActionNone        NextActionType = "none"
	NextActionCustomerPIN NextActionType = "customer_pin"
	NextActionOTPRequired NextActionType = "otp_required"
	NextActionRedirect    NextActionType = "redirect"
)

var (
	ErrProviderInvalidAccount           = errors.New("provider invalid account")
	ErrProviderUnsupportedCurrency      = errors.New("provider unsupported currency")
	ErrProviderUnsupportedCountry       = errors.New("provider unsupported country")
	ErrProviderUnsupportedPaymentMethod = errors.New("provider unsupported payment method")
	ErrProviderWebhookInvalidSignature  = errors.New("provider webhook invalid signature")
	ErrProviderWebhookUnverified        = errors.New("provider webhook unverified")
	ErrProviderPaymentNotFound          = errors.New("provider payment not found")
	ErrProviderDuplicateReference       = errors.New("provider duplicate reference")
	ErrProviderInvalidRequest           = errors.New("provider invalid request")
)

type Capabilities struct {
	Countries  []string
	Currencies []string
	Methods    []PaymentMethod

	// Direct API collection, for example PawaPay POST /v2/deposits.
	SupportsDirectCollection bool

	// Provider may return a URL as a next action in some markets.
	// This is not Leamout checkout. Leamout still owns checkout sessions.
	SupportsRedirectAction bool

	SupportsWebhook        bool
	SupportsWebhookSigning bool
	SupportsVerifyPayment  bool
}

type Customer struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Country string `json:"country,omitempty"`
}

type InitiatePaymentRequest struct {
	// Leamout owner/user/account initiating this collection.
	UserID string `json:"user_id"`

	// Leamout-generated unique payment attempt reference.
	// For PawaPay this maps to depositId.
	ExternalRef string `json:"external_ref"`

	// AmountMinor is the smallest currency unit:
	// GHS => pesewas, NGN => kobo, USD => cents.
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Country     string `json:"country"`

	Method PaymentMethod `json:"method"`

	Description string   `json:"description,omitempty"`
	Customer    Customer `json:"customer"`

	// Provider callback endpoint. Some providers configure this on dashboard
	// rather than per request, but keep it here for provider-neutral orchestration.
	CallbackURL string `json:"callback_url,omitempty"`

	// Leamout frontend return page for redirect-based authorization flows.
	ReturnURL string `json:"return_url,omitempty"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Provider-neutral metadata to help map callbacks/status checks back to Leamout.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Provider-specific configuration without polluting the generic contract.
	// PawaPay examples:
	//   provider: MTN_MOMO_GHA
	//   pre_authorisation_code: abc
	//   successful_url: https://...
	//   failed_url: https://...
	//   statement_description: Renewal payment
	ProviderOptions map[string]any `json:"provider_options,omitempty"`
}

type InitiatePaymentResponse struct {
	ProviderID ID `json:"provider_id"`

	ExternalRef       string `json:"external_ref"`
	ProviderReference string `json:"provider_reference,omitempty"`

	Status         PaymentStatus  `json:"status"`
	NextActionType NextActionType `json:"next_action_type"`
	NextActionURL  string         `json:"next_action_url,omitempty"`

	CustomerMessage string `json:"customer_message,omitempty"`

	ProviderResponse []byte            `json:"provider_response,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type VerifyPaymentRequest struct {
	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
}

type VerifyPaymentResponse struct {
	ProviderID ID `json:"provider_id"`

	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`

	Status      PaymentStatus `json:"status"`
	AmountMinor int64         `json:"amount_minor,omitempty"`
	Currency    string        `json:"currency,omitempty"`
	PaidAt      *time.Time    `json:"paid_at,omitempty"`

	ProviderResponse []byte            `json:"provider_response,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type WebhookRequest struct {
	Headers http.Header
	Body    []byte
	Path    string
}

type WebhookEvent struct {
	ProviderID ID `json:"provider_id"`

	EventID   string `json:"event_id,omitempty"`
	EventType string `json:"event_type"`

	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`

	Status PaymentStatus `json:"status"`

	Verified bool `json:"verified"`

	RawPayload []byte            `json:"raw_payload,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type Provider interface {
	ID() ID
	Name() string
	Capabilities() Capabilities
	InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*InitiatePaymentResponse, error)
	VerifyPayment(ctx context.Context, req VerifyPaymentRequest) (*VerifyPaymentResponse, error)
	ParseWebhook(ctx context.Context, req WebhookRequest) (*WebhookEvent, error)
}
