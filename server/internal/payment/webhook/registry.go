package webhook

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type Event struct {
	Provider  payment.ProviderName
	EventType string

	RawBody []byte
	Headers http.Header
	Query   url.Values
}

type ProviderHandler interface {
	Provider() payment.ProviderName
	HandleWebhook(ctx context.Context, event Event) error
}

type Registry struct {
	handlers map[payment.ProviderName]ProviderHandler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[payment.ProviderName]ProviderHandler),
	}
}

func (r *Registry) Register(handler ProviderHandler) error {
	if handler == nil {
		return fmt.Errorf("missing webhook handler")
	}

	provider := normalizeProvider(handler.Provider())
	if provider == "" {
		return fmt.Errorf("missing webhook provider name")
	}

	r.handlers[provider] = handler
	return nil
}

func (r *Registry) Get(provider payment.ProviderName) (ProviderHandler, bool) {
	if r == nil {
		return nil, false
	}

	handler, ok := r.handlers[normalizeProvider(provider)]
	return handler, ok
}

func normalizeProvider(provider payment.ProviderName) payment.ProviderName {
	return payment.ProviderName(strings.ToLower(strings.TrimSpace(string(provider))))
}
