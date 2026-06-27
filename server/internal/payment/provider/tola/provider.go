package tola

import (
	"context"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type TolaProvider struct {
	client *Client
}

func NewProvider(client *Client) *TolaProvider {
	return &TolaProvider{client: client}
}

func (p *TolaProvider) Name() payment.ProviderName {
	return payment.ProviderTola
}

func (p *TolaProvider) Charge(ctx context.Context, payload payment.UnifiedPayload) (*payment.ChargeResult, error) {
	req, err := MapChargeRequest(payload)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.CreateTransaction(ctx, req)
	if err != nil {
		return nil, err
	}

	return &payment.ChargeResult{
		TransactionID:     payload.TransactionID,
		Provider:          payment.ProviderTola,
		Status:            payment.PaymentStatusPending,
		ProviderReference: resp.Reference,
		Message:           "tola mobile charge initiated",
		CreatedAt:         time.Now().UTC(),
	}, nil
}
