package oauth

import (
	"context"
	"fmt"
)

const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Profile struct {
	Provider       string
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	AvatarURL      string
}

type Provider interface {
	Name() string
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*Profile, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{
		providers: make(map[string]Provider, len(providers)),
	}

	for _, provider := range providers {
		registry.providers[provider.Name()] = provider
	}

	return registry
}

func (r *Registry) Get(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("oauth provider %q not registered", name)
	}

	return provider, nil
}

func (r *Registry) Has(name string) bool {
	_, ok := r.providers[name]
	return ok
}
