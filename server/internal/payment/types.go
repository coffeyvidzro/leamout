package payment

import (
	"context"
	"time"
)

type ProviderName string

const (
	ProviderPawaPay ProviderName = "pawapay"
	ProviderTola    ProviderName = "tola"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
)

type UnifiedPayload struct {
	TransactionID string `json:"transactionId"`

	// Comes from checkout/user selection.
	Country string `json:"country"` // ZM, GH, KE
	Network string `json:"network"` // MTN, AIRTEL, SAFARICOM

	PhoneNumber string `json:"phoneNumber"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`

	// Filled by routing.
	Provider ProviderName `json:"provider,omitempty"`
	Operator string       `json:"operator,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

type ChargeResult struct {
	TransactionID string        `json:"transactionId"`
	Provider      ProviderName  `json:"provider"`
	Status        PaymentStatus `json:"status"`

	ProviderReference string `json:"providerReference,omitempty"`
	Message           string `json:"message,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

type ChargeProvider interface {
	Name() ProviderName
	Charge(ctx context.Context, payload UnifiedPayload) (*ChargeResult, error)
}

type Router interface {
	Resolve(ctx context.Context, payload UnifiedPayload) (*RoutingResult, error)
}

type RoutingFees struct {
	MMOFeeBps      int64 `json:"mmo_fee_bps"`
	ProviderFeeBps int64 `json:"provider_fee_bps"`
	TotalFeeBps    int64 `json:"total_fee_bps"`
}

type RoutingResult struct {
	Provider ChargeProvider
	Payload  UnifiedPayload
	Fees     RoutingFees
}
