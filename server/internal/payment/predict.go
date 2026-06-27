package payment

import (
	"context"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

func (s *Service) PredictProvider(ctx context.Context, req PredictProviderRequest) (*PredictProviderResult, error) {
	if s == nil || s.router == nil {
		return nil, ErrRouterUnavailable
	}

	phoneNumber := strings.TrimSpace(req.PhoneNumber)
	if phoneNumber == "" {
		return nil, fmt.Errorf("%w: phone_number is required", ErrInvalidRequest)
	}

	paymentProvider, ok := s.router.Provider(provider.ProviderPawaPay)
	if !ok || paymentProvider == nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, provider.ProviderPawaPay)
	}

	predictor, ok := paymentProvider.(provider.ProviderPredictor)
	if !ok || predictor == nil {
		return nil, fmt.Errorf("%w: provider %s does not support provider prediction", ErrProviderUnavailable, provider.ProviderPawaPay)
	}

	prediction, err := predictor.PredictProvider(ctx, provider.PredictProviderRequest{PhoneNumber: phoneNumber})
	if err != nil {
		return nil, err
	}
	if prediction == nil {
		return nil, fmt.Errorf("%w: provider returned nil prediction", ErrProviderUnavailable)
	}

	return &PredictProviderResult{
		Country:      strings.ToUpper(strings.TrimSpace(prediction.Country)),
		ProviderCode: strings.ToUpper(strings.TrimSpace(prediction.Provider)),
		PhoneNumber:  strings.TrimSpace(prediction.PhoneNumber),
	}, nil
}
