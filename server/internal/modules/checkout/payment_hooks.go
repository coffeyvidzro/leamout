package checkout

import (
	"context"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type PaymentHooks struct {
	repository *Repository
}

func NewPaymentHooks(repository *Repository) *PaymentHooks {
	return &PaymentHooks{repository: repository}
}

func (h *PaymentHooks) PaymentInitiated(ctx payment.Context, result *payment.InitiatePaymentResult) error {
	return nil
}

func (h *PaymentHooks) PaymentVerified(ctx payment.Context, result *payment.VerifyPaymentResult) error {
	return nil
}

func (h *PaymentHooks) WebhookProcessed(ctx payment.Context, result *payment.ProcessedWebhookResult) error {
	if h == nil || h.repository == nil || result == nil {
		return nil
	}

	providerResponse := []byte(nil)
	metadata := result.Metadata
	if result.Verification != nil {
		providerResponse = result.Verification.ProviderResponse
		if len(result.Verification.Metadata) > 0 {
			metadata = result.Verification.Metadata
		}
	}

	return h.repository.ApplyPaymentResult(contextFromPayment(ctx), ApplyPaymentResultParams{
		ExternalRef:       result.ExternalRef,
		ProviderID:        string(result.ProviderID),
		ProviderReference: result.ProviderReference,
		Status:            PaymentAttemptStatus(result.Status),
		ProviderResponse:  providerResponse,
		Metadata:          metadata,
	})
}

func contextFromPayment(ctx payment.Context) context.Context {
	if realCtx, ok := ctx.(context.Context); ok {
		return realCtx
	}
	return context.Background()
}

var _ payment.Hooks = (*PaymentHooks)(nil)
