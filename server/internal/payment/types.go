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
// The payment service maps this to each provider's expected field:
//
//   - Moolre: channel 13/6/7
//   - PawaPay: provider MTN_MOMO_GHA/VODAFONE_GHA/AIRTELTIGO_GHA
//
// Keep frontend and database values provider-neutral. Do not expose Moolre or
// PawaPay codes to the checkout page.
type MobileMoneyOperator string

const (
	MobileMoneyOperatorMTN     MobileMoneyOperator = "mtn"
	MobileMoneyOperatorTelecel MobileMoneyOperator = "telecel"
	MobileMoneyOperatorAT      MobileMoneyOperator = "at"
)

// Config controls top-level payment orchestration behavior.
type Config struct {
	// VerifyWebhookPayments should stay true in production. Webhooks are treated
	// as signals; final state is reconciled by provider.VerifyPayment.
	VerifyWebhookPayments bool

	// NormalizeCustomerPhone converts local formats such as 024xxxxxxx into a
	// provider-friendly MSISDN such as 23324xxxxxxx for Ghana.
	NormalizeCustomerPhone bool
}

func DefaultConfig() Config {
	return Config{
		VerifyWebhookPayments:  true,
		NormalizeCustomerPhone: true,
	}
}

// Customer is the checkout customer/payer. It intentionally mirrors the
// provider.Customer shape but lives in this package so modules do not need to
// import provider directly for ordinary payment initiation.
type Customer struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Country string `json:"country,omitempty"`
}

// InitiatePaymentRequest is the main input from checkout/dunning/subscription
// modules into the payment kernel.
//
// Leamout owns checkout sessions and checkout pages. Providers only initiate a
// collection request and return a normalized result.
type InitiatePaymentRequest struct {
	UserID string `json:"user_id"`

	// Leamout-generated unique payment attempt reference.
	// Create and persist this before calling InitiatePayment.
	ExternalRef string `json:"external_ref"`

	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Country     string `json:"country"`

	Method   provider.PaymentMethod `json:"method"`
	Operator MobileMoneyOperator    `json:"operator,omitempty"`

	PreferredProvider provider.ID `json:"preferred_provider,omitempty"`

	Description string   `json:"description,omitempty"`
	Customer    Customer `json:"customer"`

	CallbackURL string `json:"callback_url,omitempty"`
	ReturnURL   string `json:"return_url,omitempty"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	Metadata        map[string]string `json:"metadata,omitempty"`
	ProviderOptions map[string]any    `json:"provider_options,omitempty"`
}

// InitiatePaymentResult is the normalized provider result plus route context.
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

// VerifyPaymentRequest asks a specific provider to reconcile a payment attempt.
type VerifyPaymentRequest struct {
	ProviderID provider.ID `json:"provider_id"`

	ExternalRef       string `json:"external_ref,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
}

// VerifyPaymentResult is the normalized status returned by provider verification.
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

// ProcessedWebhookResult is returned by ProcessWebhookEvent and is useful in
// tests, workers, and future audit logging.
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

// RouteInfo is a serializable copy of routing.RouteResult without the provider
// instance field.
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

// Hooks let the surrounding application persist attempts/events or trigger
// domain actions without making this package depend on checkout, subscription,
// dunning, or database packages.
type Hooks interface {
	PaymentInitiated(ctx Context, result *InitiatePaymentResult) error
	PaymentVerified(ctx Context, result *VerifyPaymentResult) error
	WebhookProcessed(ctx Context, result *ProcessedWebhookResult) error
}

// Context is an alias-friendly interface satisfied by context.Context.
// It avoids forcing hook implementations to import a custom type.
type Context interface {
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

// NoopHooks can be embedded in app-level hooks so only needed callbacks are
// implemented.
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
	case "at", "airteltigo", "airtel_tigo", "at_money", "tigo":
		return MobileMoneyOperatorAT
	default:
		return MobileMoneyOperator(raw)
	}
}
