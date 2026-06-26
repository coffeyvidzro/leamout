package pawapay

import (
	"context"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

type PawapayProvider struct {
	client *Client
}

func NewPawapayProvider(client *Client) *PawapayProvider {
	return &PawapayProvider{client: client}
}

func (p *PawapayProvider) ID() provider.ID {
	return provider.ProviderPawaPay
}

func (p *PawapayProvider) Name() string {
	return "PawaPay"
}

func (p *PawapayProvider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		// Leamout MVP: Ghana mobile money collections.
		// Use active-conf later to discover the exact enabled providers
		// on the merchant account.
		Countries:  []string{"GH", "GHA"},
		Currencies: []string{"GHS"},
		Methods: []provider.PaymentMethod{
			provider.PaymentMethodMobileMoney,
		},

		SupportsDirectCollection: true,
		SupportsRedirectAction:   true,
		SupportsWebhook:          true,
		SupportsWebhookSigning:   true,
		SupportsVerifyPayment:    true,
	}
}

func (p *PawapayProvider) InitiatePayment(ctx context.Context, req provider.InitiatePaymentRequest) (*provider.InitiatePaymentResponse, error) {
	if p.client == nil {
		return nil, provider.ErrProviderInvalidAccount
	}

	if err := p.validateRequest(req); err != nil {
		return nil, err
	}

	pawaReq, err := FromInternal(req)
	if err != nil {
		return nil, err
	}

	pawaResp, raw, err := p.client.InitiateDeposit(ctx, pawaReq)
	if err != nil {
		return nil, fmt.Errorf("pawapay initiate deposit failed: %w", err)
	}

	resp := ToInitiateResponse(pawaResp, raw)
	if resp.ExternalRef == "" {
		resp.ExternalRef = req.ExternalRef
	}
	if resp.ProviderReference == "" {
		resp.ProviderReference = req.ExternalRef
	}

	return resp, nil
}

func (p *PawapayProvider) VerifyPayment(ctx context.Context, req provider.VerifyPaymentRequest) (*provider.VerifyPaymentResponse, error) {
	if p.client == nil {
		return nil, provider.ErrProviderInvalidAccount
	}

	depositID := strings.TrimSpace(req.ExternalRef)
	if depositID == "" {
		depositID = strings.TrimSpace(req.ProviderReference)
	}
	if depositID == "" {
		return nil, fmt.Errorf("%w: external_ref or provider_reference is required", provider.ErrProviderInvalidRequest)
	}

	statusResp, raw, err := p.client.CheckDepositStatus(ctx, depositID)
	if err != nil {
		return nil, fmt.Errorf("pawapay check deposit status failed: %w", err)
	}

	return ToVerifyResponse(statusResp, depositID, raw), nil
}

func (p *PawapayProvider) ParseWebhook(ctx context.Context, req provider.WebhookRequest) (*provider.WebhookEvent, error) {
	return ParseWebhook(ctx, req)
}

func (p *PawapayProvider) validateRequest(req provider.InitiatePaymentRequest) error {
	if strings.TrimSpace(req.Country) != "" {
		country := strings.ToUpper(strings.TrimSpace(req.Country))
		if country != "GH" && country != "GHA" {
			return provider.ErrProviderUnsupportedCountry
		}
	}

	if strings.ToUpper(strings.TrimSpace(req.Currency)) != "GHS" {
		return provider.ErrProviderUnsupportedCurrency
	}

	if req.Method != "" && req.Method != provider.PaymentMethodMobileMoney {
		return provider.ErrProviderUnsupportedPaymentMethod
	}

	return nil
}
