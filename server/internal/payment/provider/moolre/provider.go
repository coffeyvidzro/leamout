package moolre

import (
	"context"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

type Provider struct {
	client *Client
}

func NewProvider(client *Client) *Provider {
	return &Provider{client: client}
}

func NewProviderFromConfig(cfg Config) *Provider {
	return NewProvider(NewClient(cfg))
}

func (p *Provider) ID() provider.ID {
	return provider.ProviderMoolre
}

func (p *Provider) Name() string {
	return "Moolre"
}

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Countries:  []string{"GH"},
		Currencies: []string{"GHS"},
		Methods: []provider.PaymentMethod{
			provider.PaymentMethodMobileMoney,
		},
		SupportsDirectCollection: true,
		SupportsRedirectAction:   false,
		SupportsWebhook:          true,
		SupportsWebhookSigning:   false,
		SupportsVerifyPayment:    true,
	}
}

func (p *Provider) InitiatePayment(ctx context.Context, req provider.InitiatePaymentRequest) (*provider.InitiatePaymentResponse, error) {
	return p.client.InitiatePayment(ctx, req)
}

func (p *Provider) VerifyPayment(ctx context.Context, req provider.VerifyPaymentRequest) (*provider.VerifyPaymentResponse, error) {
	return p.client.VerifyPayment(ctx, req)
}

func (p *Provider) ParseWebhook(ctx context.Context, req provider.WebhookRequest) (*provider.WebhookEvent, error) {
	return ParseWebhook(ctx, req)
}
