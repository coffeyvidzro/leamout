package pawapay

import (
	"context"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
	paymentregistry "github.com/cuffeyvidzro/leamout/internal/payment/registry"
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
	rules := paymentregistry.PawaPayMVPRules()
	countries := make([]string, 0, len(rules)*2)
	currencies := make([]string, 0, len(rules))

	for _, rule := range rules {
		countries = append(countries, rule.Country, rule.CountryAlpha3)
		currencies = append(currencies, rule.Currency)
	}

	return provider.Capabilities{
		Countries:  uniqueStrings(countries),
		Currencies: uniqueStrings(currencies),
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

func (p *PawapayProvider) PredictProvider(ctx context.Context, req provider.PredictProviderRequest) (*provider.PredictProviderResponse, error) {
	if p.client == nil {
		return nil, provider.ErrProviderInvalidAccount
	}

	phoneNumber := strings.TrimSpace(req.PhoneNumber)
	if phoneNumber == "" {
		return nil, fmt.Errorf("%w: phone_number is required", provider.ErrProviderInvalidRequest)
	}

	prediction, _, err := p.client.PredictProvider(ctx, phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("pawapay predict provider failed: %w", err)
	}
	if prediction == nil {
		return nil, fmt.Errorf("%w: pawapay returned nil provider prediction", provider.ErrProviderInvalidRequest)
	}

	return &provider.PredictProviderResponse{
		Country:     strings.ToUpper(strings.TrimSpace(prediction.Country)),
		Provider:    strings.ToUpper(strings.TrimSpace(prediction.Provider)),
		PhoneNumber: strings.TrimSpace(prediction.PhoneNumber),
	}, nil
}

func (p *PawapayProvider) ParseWebhook(ctx context.Context, req provider.WebhookRequest) (*provider.WebhookEvent, error) {
	return ParseWebhook(ctx, req)
}

func (p *PawapayProvider) validateRequest(req provider.InitiatePaymentRequest) error {
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	expectedCurrency, ok := supportedCurrencyForCountry(country)
	if country != "" && !ok {
		return provider.ErrProviderUnsupportedCountry
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		return provider.ErrProviderUnsupportedCurrency
	}
	if expectedCurrency != "" && currency != expectedCurrency {
		return provider.ErrProviderUnsupportedCurrency
	}

	if req.Method != "" && req.Method != provider.PaymentMethodMobileMoney {
		return provider.ErrProviderUnsupportedPaymentMethod
	}

	return nil
}

func supportedCurrencyForCountry(country string) (string, bool) {
	country = strings.ToUpper(strings.TrimSpace(country))
	for _, rule := range paymentregistry.PawaPayMVPRules() {
		if rule.Country == country || rule.CountryAlpha3 == country {
			return rule.Currency, true
		}
	}
	return "", false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

var _ provider.ProviderPredictor = (*PawapayProvider)(nil)
