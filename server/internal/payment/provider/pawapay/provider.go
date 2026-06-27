package pawapay

import (
	"context"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

var supportedCountryCurrencies = map[string]string{
	"BJ":  "XOF",
	"BEN": "XOF",
	"BF":  "XOF",
	"BFA": "XOF",
	"CI":  "XOF",
	"CIV": "XOF",
	"CM":  "XAF",
	"CMR": "XAF",
	"CD":  "CDF",
	"COD": "CDF",
	"CG":  "XAF",
	"COG": "XAF",
	"GA":  "XAF",
	"GAB": "XAF",
	"GH":  "GHS",
	"GHA": "GHS",
	"MW":  "MWK",
	"MWI": "MWK",
	"RW":  "RWF",
	"RWA": "RWF",
	"SN":  "XOF",
	"SEN": "XOF",
	"SL":  "SLE",
	"SLE": "SLE",
	"TZ":  "TZS",
	"TZA": "TZS",
	"UG":  "UGX",
	"UGA": "UGX",
}

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
		// Leamout MVP uses PawaPay as the only aggregator. Kenya, Mozambique,
		// and Zambia are intentionally excluded until their fee rules are ready.
		Countries:  []string{"BJ", "BEN", "BF", "BFA", "CI", "CIV", "CM", "CMR", "CD", "COD", "CG", "COG", "GA", "GAB", "GH", "GHA", "MW", "MWI", "RW", "RWA", "SN", "SEN", "SL", "SLE", "TZ", "TZA", "UG", "UGA"},
		Currencies: []string{"CDF", "GHS", "MWK", "RWF", "SLE", "TZS", "UGX", "XAF", "XOF"},
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
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	expectedCurrency, ok := supportedCountryCurrencies[country]
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
