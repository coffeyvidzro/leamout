package pawapay

import (
	"context"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type PawapayProvider struct {
	client *Client
}

func NewProvider(client *Client) *PawapayProvider {
	return &PawapayProvider{client: client}
}

func (p *PawapayProvider) Name() payment.ProviderName {
	return payment.ProviderPawaPay
}

func (p *PawapayProvider) Charge(ctx context.Context, payload payment.UnifiedPayload) (*payment.ChargeResult, error) {
	req, err := MapDepositRequest(payload)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.CreateDeposit(ctx, req)
	if err != nil {
		return nil, err
	}

	return &payment.ChargeResult{
		TransactionID:     payload.TransactionID,
		Provider:          payment.ProviderPawaPay,
		Status:            payment.PaymentStatusPending,
		ProviderReference: resp.DepositID,
		Message:           "pawapay deposit initiated",
		CreatedAt:         time.Now().UTC(),
	}, nil
}
