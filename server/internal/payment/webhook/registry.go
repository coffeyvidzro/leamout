package webhook

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

var (
	ErrNilProvider           = errors.New("payment webhook provider is nil")
	ErrProviderIDEmpty       = errors.New("payment webhook provider id is empty")
	ErrProviderAlreadyExists = errors.New("payment webhook provider already registered")
	ErrProviderNotFound      = errors.New("payment webhook provider not registered")
)

// Registry stores payment providers that can parse their own webhook payloads.
//
// The webhook package must stay provider-neutral: it should know that a provider
// exists, but it must not import moolre, pawapay, hubtel, etc. Provider-specific
// JSON parsing and signature verification live inside each provider adapter.
type Registry struct {
	mu        sync.RWMutex
	providers map[provider.ID]provider.Provider
}

func NewRegistry(providers ...provider.Provider) (*Registry, error) {
	r := &Registry{
		providers: make(map[provider.ID]provider.Provider),
	}

	for _, p := range providers {
		if err := r.Register(p); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func MustNewRegistry(providers ...provider.Provider) *Registry {
	r, err := NewRegistry(providers...)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *Registry) Register(p provider.Provider) error {
	if p == nil {
		return ErrNilProvider
	}

	id := NormalizeProviderID(p.ID())
	if id == "" {
		return ErrProviderIDEmpty
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("%w: %s", ErrProviderAlreadyExists, id)
	}

	r.providers[id] = p
	return nil
}

func (r *Registry) Unregister(id provider.ID) bool {
	id = NormalizeProviderID(id)
	if id == "" {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; !exists {
		return false
	}

	delete(r.providers, id)
	return true
}

func (r *Registry) Get(id provider.ID) (provider.Provider, bool) {
	id = NormalizeProviderID(id)
	if id == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[id]
	return p, ok
}

func (r *Registry) MustGet(id provider.ID) (provider.Provider, error) {
	id = NormalizeProviderID(id)

	p, ok := r.Get(id)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, id)
	}

	return p, nil
}

func (r *Registry) Has(id provider.ID) bool {
	_, ok := r.Get(id)
	return ok
}

func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers)
}

func (r *Registry) IDs() []provider.ID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]provider.ID, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return ids
}

func NormalizeProviderID(id provider.ID) provider.ID {
	return provider.ID(strings.ToLower(strings.TrimSpace(string(id))))
}
