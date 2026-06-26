package routing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

var (
	ErrNoProviderAvailable = errors.New("no payment provider available")
	ErrNoProviderMatched   = errors.New("no payment provider matched route")
)

// RouteRequest is the provider-neutral input used to choose a payment provider.
type RouteRequest struct {
	Country  string                 `json:"country"`
	Currency string                 `json:"currency"`
	Method   provider.PaymentMethod `json:"method"`

	// PreferredProvider is tried first, but routing can fall back when allowed.
	PreferredProvider provider.ID `json:"preferred_provider,omitempty"`

	// AmountMinor is optional for today's routing but useful when you later route
	// by provider limits, fees, or success rate bands.
	AmountMinor int64 `json:"amount_minor,omitempty"`
}

// RouteResult contains the selected provider and the reasoning trail.
type RouteResult struct {
	ProviderID provider.ID       `json:"provider_id"`
	Provider   provider.Provider `json:"-"`

	RouteKey        string        `json:"route_key"`
	SelectionReason string        `json:"selection_reason"`
	CandidateIDs    []provider.ID `json:"candidate_ids"`
	Skipped         []SkipReason  `json:"skipped,omitempty"`
}

type SkipReason struct {
	ProviderID provider.ID `json:"provider_id"`
	Reason     string      `json:"reason"`
}

type Strategy interface {
	Resolve(ctx context.Context, cfg Config, providers []provider.Provider, req RouteRequest) (*RouteResult, error)
}

// StaticStrategy chooses the first configured provider that is enabled,
// registered, and capable of handling the route.
type StaticStrategy struct{}

func NewStaticStrategy() *StaticStrategy {
	return &StaticStrategy{}
}

func (s *StaticStrategy) Resolve(ctx context.Context, cfg Config, providers []provider.Provider, req RouteRequest) (*RouteResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cfg = cfg.normalized()
	req = req.normalized()

	if len(providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	available := providerMap(providers)
	if len(available) == 0 {
		return nil, ErrNoProviderAvailable
	}

	candidateIDs, matchedRoute := buildCandidateList(cfg, req)
	result := &RouteResult{
		RouteKey:     routeKey(req.Country, req.Currency, req.Method),
		CandidateIDs: candidateIDs,
		Skipped:      make([]SkipReason, 0),
	}

	chosen, reason, ok := firstUsableProvider(cfg, req, available, candidateIDs, &result.Skipped)
	if ok {
		result.ProviderID = chosen.ID()
		result.Provider = chosen
		result.SelectionReason = reason
		return result, nil
	}

	if cfg.AllowFallback {
		fallbackIDs := fallbackCandidateIDs(cfg.EnabledProviders, candidateIDs)
		if len(fallbackIDs) > 0 {
			result.CandidateIDs = append(result.CandidateIDs, fallbackIDs...)
			chosen, reason, ok = firstUsableProvider(cfg, req, available, fallbackIDs, &result.Skipped)
			if ok {
				result.ProviderID = chosen.ID()
				result.Provider = chosen
				result.SelectionReason = reason
				return result, nil
			}
		}
	}

	if matchedRoute {
		return result, fmt.Errorf("%w for %s", ErrNoProviderMatched, result.RouteKey)
	}
	return result, fmt.Errorf("%w for %s", ErrNoProviderMatched, result.RouteKey)
}

func (r RouteRequest) normalized() RouteRequest {
	out := r
	out.Country = normalizeCountry(out.Country)
	out.Currency = normalizeCurrency(out.Currency)
	out.Method = normalizeMethod(out.Method)
	out.PreferredProvider = normalizeProviderID(string(out.PreferredProvider))
	return out
}

func buildCandidateList(cfg Config, req RouteRequest) ([]provider.ID, bool) {
	ids := make([]provider.ID, 0)

	if req.PreferredProvider != "" {
		ids = append(ids, req.PreferredProvider)
	}

	matchedRoute := false
	if route, ok := cfg.RouteFor(req); ok {
		matchedRoute = true
		ids = append(ids, route.Providers...)
	} else if cfg.DefaultProvider != "" {
		ids = append(ids, cfg.DefaultProvider)
	}

	if len(ids) == 0 {
		ids = append(ids, cfg.EnabledProviders...)
	}

	return dedupeProviderIDs(ids), matchedRoute
}

func fallbackCandidateIDs(enabled []provider.ID, alreadyTried []provider.ID) []provider.ID {
	tried := make(map[provider.ID]struct{}, len(alreadyTried))
	for _, id := range alreadyTried {
		tried[normalizeProviderID(string(id))] = struct{}{}
	}

	ids := make([]provider.ID, 0)
	for _, id := range enabled {
		id = normalizeProviderID(string(id))
		if id == "" {
			continue
		}
		if _, ok := tried[id]; ok {
			continue
		}
		ids = append(ids, id)
	}
	return dedupeProviderIDs(ids)
}

func firstUsableProvider(
	cfg Config,
	req RouteRequest,
	available map[provider.ID]provider.Provider,
	candidateIDs []provider.ID,
	skipped *[]SkipReason,
) (provider.Provider, string, bool) {
	for _, id := range candidateIDs {
		id = normalizeProviderID(string(id))
		if id == "" {
			continue
		}

		if !cfg.IsProviderEnabled(id) {
			appendSkip(skipped, id, "provider is disabled")
			continue
		}

		p, ok := available[id]
		if !ok {
			appendSkip(skipped, id, "provider is not registered")
			continue
		}

		if cfg.StrictCapabilities {
			if ok, reason := providerSupportsRequest(p, req); !ok {
				appendSkip(skipped, id, reason)
				continue
			}
		}

		return p, "selected first enabled registered provider that supports route", true
	}

	return nil, "", false
}

func appendSkip(skipped *[]SkipReason, id provider.ID, reason string) {
	*skipped = append(*skipped, SkipReason{ProviderID: id, Reason: reason})
}

func providerMap(providers []provider.Provider) map[provider.ID]provider.Provider {
	out := make(map[provider.ID]provider.Provider, len(providers))
	for _, p := range providers {
		if p == nil {
			continue
		}
		id := normalizeProviderID(string(p.ID()))
		if id == "" {
			continue
		}
		out[id] = p
	}
	return out
}

func providerSupportsRequest(p provider.Provider, req RouteRequest) (bool, string) {
	caps := p.Capabilities()
	id := p.ID()

	if !caps.SupportsDirectCollection {
		return false, fmt.Sprintf("provider %q does not support direct collection", id)
	}

	if len(caps.Countries) > 0 && !containsStringFold(caps.Countries, req.Country) {
		return false, fmt.Sprintf("provider %q does not support country %s", id, req.Country)
	}

	if len(caps.Currencies) > 0 && !containsStringFold(caps.Currencies, req.Currency) {
		return false, fmt.Sprintf("provider %q does not support currency %s", id, req.Currency)
	}

	if len(caps.Methods) > 0 && !containsPaymentMethod(caps.Methods, req.Method) {
		return false, fmt.Sprintf("provider %q does not support payment method %s", id, req.Method)
	}

	return true, ""
}

func containsStringFold(items []string, target string) bool {
	target = strings.ToUpper(strings.TrimSpace(target))
	for _, item := range items {
		if strings.ToUpper(strings.TrimSpace(item)) == target {
			return true
		}
	}
	return false
}

func containsPaymentMethod(items []provider.PaymentMethod, target provider.PaymentMethod) bool {
	target = normalizeMethod(target)
	for _, item := range items {
		if normalizeMethod(item) == target {
			return true
		}
	}
	return false
}
