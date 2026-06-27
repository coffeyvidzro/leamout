package payment

import (
	"context"
	"errors"
	"strings"

	paymentkernel "github.com/cuffeyvidzro/leamout/internal/payment"
)

type ProviderPrediction struct {
	Country      string
	ProviderCode string
	PhoneNumber  string
}

type processorLookup interface {
	PredictProvider(ctx context.Context, req paymentkernel.PredictProviderRequest) (*paymentkernel.PredictProviderResult, error)
}

func (s *Service) PredictCheckoutProvider(ctx context.Context, phoneNumber string) (*ProviderPrediction, error) {
	if s.processor == nil {
		return nil, errors.New("payment processor is not configured")
	}

	phoneNumber = strings.TrimSpace(phoneNumber)
	if phoneNumber == "" {
		return nil, ErrInvalidPayment
	}

	lookup, ok := s.processor.(processorLookup)
	if !ok || lookup == nil {
		return nil, errors.New("payment processor does not support provider lookup")
	}

	prediction, err := lookup.PredictProvider(ctx, paymentkernel.PredictProviderRequest{PhoneNumber: phoneNumber})
	if err != nil {
		return nil, err
	}
	if prediction == nil {
		return nil, ErrInvalidPayment
	}

	return &ProviderPrediction{Country: prediction.Country, ProviderCode: prediction.ProviderCode, PhoneNumber: prediction.PhoneNumber}, nil
}
