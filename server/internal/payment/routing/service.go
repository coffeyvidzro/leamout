package routing

import (
	"context"
	"fmt"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type Service struct {
	config    *Config
	strategy  Strategy
	providers map[payment.ProviderName]payment.ChargeProvider
}

func NewService(
	cfg *Config,
	strategy Strategy,
	providers map[payment.ProviderName]payment.ChargeProvider,
) *Service {
	if cfg == nil {
		cfg = NewDefaultConfig()
	}

	if strategy == nil {
		strategy = NewPriorityStrategy()
	}

	if providers == nil {
		providers = make(map[payment.ProviderName]payment.ChargeProvider)
	}

	return &Service{
		config:    cfg,
		strategy:  strategy,
		providers: providers,
	}
}

func (s *Service) Resolve(ctx context.Context, payload payment.UnifiedPayload) (*payment.RoutingResult, error) {
	_ = ctx

	routes, err := s.config.Lookup(payload.Country, payload.Network, payload.Currency)
	if err != nil {
		return nil, err
	}

	selectedRoute, err := s.strategy.Select(routes)
	if err != nil {
		return nil, fmt.Errorf("select payment route: %w", err)
	}

	selectedProvider, ok := s.providers[selectedRoute.Provider]
	if !ok || selectedProvider == nil {
		return nil, fmt.Errorf("payment provider %s is not registered", selectedRoute.Provider)
	}

	routedPayload := payload
	routedPayload.Provider = selectedRoute.Provider
	routedPayload.Operator = selectedRoute.Operator
	routedPayload.Country = selectedRoute.Country
	routedPayload.Network = selectedRoute.Network
	routedPayload.Currency = selectedRoute.Currency

	return &payment.RoutingResult{
		Provider: selectedProvider,
		Payload:  routedPayload,
	}, nil
}
