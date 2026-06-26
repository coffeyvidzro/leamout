package routing

import (
	"context"
	"fmt"
	"sync"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

// Service owns runtime provider routing. It does not initiate payments itself;
// it only selects the best provider for the payment service to call.
type Service struct {
	mu sync.RWMutex

	cfg       Config
	strategy  Strategy
	providers map[provider.ID]provider.Provider
}

func NewService(cfg Config, strategy Strategy, providers ...provider.Provider) (*Service, error) {
	cfg = cfg.normalized()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if strategy == nil {
		strategy = NewStaticStrategy()
	}

	svc := &Service{
		cfg:       cfg,
		strategy:  strategy,
		providers: make(map[provider.ID]provider.Provider),
	}

	for _, p := range providers {
		if err := svc.RegisterProvider(p); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

func NewServiceFromEnv(strategy Strategy, providers ...provider.Provider) (*Service, error) {
	return NewService(LoadConfigFromEnv(), strategy, providers...)
}

func (s *Service) RegisterProvider(p provider.Provider) error {
	if p == nil {
		return fmt.Errorf("payment routing provider is nil")
	}

	id := normalizeProviderID(string(p.ID()))
	if id == "" {
		return fmt.Errorf("payment routing provider id is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[id]; exists {
		return fmt.Errorf("payment routing provider %q already registered", id)
	}

	s.providers[id] = p
	return nil
}

func (s *Service) UnregisterProvider(id provider.ID) {
	id = normalizeProviderID(string(id))

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.providers, id)
}

func (s *Service) Resolve(ctx context.Context, req RouteRequest) (*RouteResult, error) {
	s.mu.RLock()
	cfg := s.cfg
	strategy := s.strategy
	providers := s.providerListLocked()
	s.mu.RUnlock()

	if strategy == nil {
		strategy = NewStaticStrategy()
	}

	return strategy.Resolve(ctx, cfg, providers, req)
}

func (s *Service) ResolveProvider(ctx context.Context, req RouteRequest) (provider.Provider, *RouteResult, error) {
	result, err := s.Resolve(ctx, req)
	if err != nil {
		return nil, result, err
	}
	return result.Provider, result, nil
}

func (s *Service) Provider(id provider.ID) (provider.Provider, bool) {
	id = normalizeProviderID(string(id))

	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.providers[id]
	return p, ok
}

func (s *Service) Providers() []provider.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.providerListLocked()
}

func (s *Service) ProviderIDs() []provider.ID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]provider.ID, 0, len(s.providers))
	for id := range s.providers {
		ids = append(ids, id)
	}
	return dedupeProviderIDs(ids)
}

func (s *Service) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Service) UpdateConfig(cfg Config) error {
	cfg = cfg.normalized()
	if err := cfg.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

func (s *Service) providerListLocked() []provider.Provider {
	providers := make([]provider.Provider, 0, len(s.providers))
	for _, p := range s.providers {
		providers = append(providers, p)
	}
	return providers
}
