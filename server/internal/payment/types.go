package payment

import (
	"errors"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

var (
	ErrInvalidRequest      = errors.New("payment invalid request")
	ErrRouterUnavailable   = errors.New("payment router unavailable")
	ErrProviderUnavailable = errors.New("payment provider unavailable")
	ErrVerificationFailed  = errors.New("payment verification failed")
)

// MobileMoneyOperator is Leamout's provider-neutral operator/network value.
// Keep frontend and database values provider-neutral. Do not expose PawaPay
// provider codes to the checkout page.
type MobileMoneyOperator string

const (
	MobileMoneyOperatorMTN     MobileMoneyOperator = "mtn"
	MobileMoneyOperatorTelecel MobileMoneyOperator = "telecel"
	MobileMoneyOperatorAT      MobileMoneyOperator = "at"
)

type Config struct {
	VerifyWebhookPayments bool
	NormalizeCustomerPhone bool
}

func DefaultConfig() Config {
	return Config{VerifyWebhookPayments: true, NormalizeCustomerPhone: true}
}

type Customer struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Country string `json:"country,omitempty"`
}

type PredictProviderRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type PredictProviderResult struct {
	Country      string `json:"country"`
	ProviderCode string `json:"provider_code"`
	PhoneNumber  string `json:"phone_number"`
}

type InitiatePaymentRequest struct {
	UserID string `json:"user_id"`
	ExternalRef string `json:"external_ref"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Country     string `json:"country"`
	Method   provider.PaymentMethod `json:"method"`
	Operator MobileMoneyOperator    `json:"operator,omitempty"`
	// Kept for kernel compatibility only. Checkout no longer accepts provider selection in the MVP.
	PreferredProvider provider.ID `json:"preferred_provider,omitempty"`
	Description string   `json:"description,omitempty"`
	Customer    Customer `json:"customer"`
	CallbackURL string `json:"callback_url,omitempty"`
	ReturnURL   string `json:"return_url,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ProviderOptions map[string]any    `json:"provider_options,omitempty"`
}

type InitiatePaymentResult struct {
	ProviderID   provider.ID `json:"provider_id"`
	ProviderName string      `json:"provider_name"`
	ExternalRef       string `json:"external_ref"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Status         provider.PaymentStatus  `json:"status"`
	NextActionType provider.NextActionType `json:"next_action_type"`
	NextActionURL  string                  `json:"next_action_url,omitempty"`
	CustomerMessage string `json:"customer_message,omitempty"`
	Route RouteInfo `json:"route"`
	ProviderResponse []byte            `json:"provider_response,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type VerifyPaymentRequest struct {
	ProviderID provider.ID `json:"provider_id"`
	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
}

type VerifyPaymentResult struct {
	ProviderID provider.ID `json:"provider_id"`
	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Status      provider.PaymentStatus `json:"status"`
	AmountMinor int64                  `json:"amount_minor,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	PaidAt      *time.Time             `json:"paid_at,omitempty"`
	ProviderResponse []byte            `json:"provider_response,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type ProcessedWebhookResult struct {
	ProviderID provider.ID `json:"provider_id"`
	EventType string                 `json:"event_type"`
	Status    provider.PaymentStatus `json:"status"`
	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Verified          bool   `json:"verified"`
	Verification *VerifyPaymentResult `json:"verification,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
}

type RouteInfo struct {
	ProviderID      provider.ID   `json:"provider_id"`
	RouteKey        string        `json:"route_key"`
	SelectionReason string        `json:"selection_reason,omitempty"`
	CandidateIDs    []provider.ID `json:"candidate_ids,omitempty"`
	Skipped         []RouteSkip   `json:"skipped,omitempty"`
}

type RouteSkip struct {
	ProviderID provider.ID `json:"provider_id"`
	Reason     string      `json:"reason"`
}

type Hooks interface {
	PaymentInitiated(ctx Context, result *InitiatePaymentResult) error
	PaymentVerified(ctx Context, result *VerifyPaymentResult) error
	WebhookProcessed(ctx Context, result *ProcessedWebhookResult) error
}

type Context interface {
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type NoopHooks struct{}

func (NoopHooks) PaymentInitiated(Context, *InitiatePaymentResult) error  { return nil }
func (NoopHooks) PaymentVerified(Context, *VerifyPaymentResult) error     { return nil }
func (NoopHooks) WebhookProcessed(Context, *ProcessedWebhookResult) error { return nil }

func normalizeOperator(operator MobileMoneyOperator) MobileMoneyOperator {
	raw := strings.ToLower(strings.TrimSpace(string(operator)))
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.ReplaceAll(raw, "-", "_")

	switch raw {
	case "mtn", "mtn_momo", "mtn_mobile_money":
		return MobileMoneyOperatorMTN
	case "telecel", "telecel_cash", "vodafone", "vodafone_cash":
		return MobileMoneyOperatorTelecel
	case "at", "airteltigo", "airtel_tigo", "at_money":
		return MobileMoneyOperatorAT
	default:
		return MobileMoneyOperator(raw)
	}
}
