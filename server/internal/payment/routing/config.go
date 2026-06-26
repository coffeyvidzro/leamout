package routing

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

const (
	envEnabledProviders   = "PAYMENT_ENABLED_PROVIDERS"
	envDefaultProvider    = "PAYMENT_DEFAULT_PROVIDER"
	envAllowFallback      = "PAYMENT_ROUTING_ALLOW_FALLBACK"
	envStrictCapabilities = "PAYMENT_ROUTING_STRICT_CAPABILITIES"
	envRoutePrefix        = "PAYMENT_ROUTE_"
)

// Config controls how Leamout chooses a payment provider for a payment attempt.
//
// Routes are prioritized. The first provider that is enabled, registered, and
// capable of serving the request is selected.
//
// Example environment route:
// PAYMENT_ROUTE_GH_GHS_MOBILE_MONEY=moolre,pawapay
//
// The suffix format is:
// PAYMENT_ROUTE_<COUNTRY>_<CURRENCY>_<METHOD>
// where METHOD may contain underscores, such as MOBILE_MONEY.
type Config struct {
	EnabledProviders   []provider.ID `json:"enabled_providers"`
	DefaultProvider    provider.ID   `json:"default_provider,omitempty"`
	Routes             []Route       `json:"routes"`
	AllowFallback      bool          `json:"allow_fallback"`
	StrictCapabilities bool          `json:"strict_capabilities"`
}

// Route declares provider priority for a specific market/payment method.
type Route struct {
	Country   string                 `json:"country"`
	Currency  string                 `json:"currency"`
	Method    provider.PaymentMethod `json:"method"`
	Providers []provider.ID          `json:"providers"`
}

// DefaultConfig gives Leamout a Ghana-first configuration while keeping
// PawaPay available as a fallback.
func DefaultConfig() Config {
	return Config{
		EnabledProviders: []provider.ID{
			provider.ProviderMoolre,
			provider.ProviderPawaPay,
		},
		DefaultProvider: provider.ProviderMoolre,
		Routes: []Route{
			{
				Country:  "GH",
				Currency: "GHS",
				Method:   provider.PaymentMethodMobileMoney,
				Providers: []provider.ID{
					provider.ProviderMoolre,
					provider.ProviderPawaPay,
				},
			},
		},
		AllowFallback:      true,
		StrictCapabilities: true,
	}
}

// LoadConfigFromEnv loads routing config from the process environment.
func LoadConfigFromEnv() Config {
	return ConfigFromEnv(os.Environ())
}

// ConfigFromEnv loads routing config from an env slice, which makes it easy to test.
func ConfigFromEnv(environ []string) Config {
	cfg := DefaultConfig()
	env := envMap(environ)

	if raw := strings.TrimSpace(env[envEnabledProviders]); raw != "" {
		cfg.EnabledProviders = parseProviderList(raw)
	}

	if raw := strings.TrimSpace(env[envDefaultProvider]); raw != "" {
		cfg.DefaultProvider = normalizeProviderID(raw)
	}

	cfg.AllowFallback = parseBoolDefault(env[envAllowFallback], cfg.AllowFallback)
	cfg.StrictCapabilities = parseBoolDefault(env[envStrictCapabilities], cfg.StrictCapabilities)

	configuredRoutes := make(map[string]Route)
	for _, route := range cfg.Routes {
		route = route.normalized()
		configuredRoutes[route.key()] = route
	}

	for key, value := range env {
		if !strings.HasPrefix(key, envRoutePrefix) {
			continue
		}

		route, ok := parseRouteEnv(key, value)
		if !ok {
			continue
		}
		configuredRoutes[route.key()] = route
	}

	cfg.Routes = routesFromMap(configuredRoutes)
	return cfg.normalized()
}

func (c Config) normalized() Config {
	out := c
	out.EnabledProviders = normalizeProviderIDs(out.EnabledProviders)
	out.DefaultProvider = normalizeProviderID(string(out.DefaultProvider))

	out.Routes = make([]Route, 0, len(c.Routes))
	for _, route := range c.Routes {
		route = route.normalized()
		if route.Country == "" || route.Currency == "" || route.Method == "" || len(route.Providers) == 0 {
			continue
		}
		out.Routes = append(out.Routes, route)
	}

	sort.SliceStable(out.Routes, func(i, j int) bool {
		return out.Routes[i].key() < out.Routes[j].key()
	})

	return out
}

func (c Config) Validate() error {
	if len(c.EnabledProviders) == 0 {
		return fmt.Errorf("payment routing has no enabled providers")
	}

	if c.DefaultProvider != "" && !c.IsProviderEnabled(c.DefaultProvider) {
		return fmt.Errorf("default payment provider %q is not enabled", c.DefaultProvider)
	}

	for _, route := range c.Routes {
		route = route.normalized()
		if route.Country == "" {
			return fmt.Errorf("payment route country is empty")
		}
		if route.Currency == "" {
			return fmt.Errorf("payment route currency is empty")
		}
		if route.Method == "" {
			return fmt.Errorf("payment route method is empty")
		}
		if len(route.Providers) == 0 {
			return fmt.Errorf("payment route %s has no providers", route.key())
		}
		for _, id := range route.Providers {
			if !c.IsProviderEnabled(id) {
				return fmt.Errorf("payment route %s uses disabled provider %q", route.key(), id)
			}
		}
	}

	return nil
}

func (c Config) IsProviderEnabled(id provider.ID) bool {
	id = normalizeProviderID(string(id))
	if id == "" {
		return false
	}
	for _, enabled := range c.EnabledProviders {
		if normalizeProviderID(string(enabled)) == id {
			return true
		}
	}
	return false
}

func (c Config) RouteFor(req RouteRequest) (Route, bool) {
	req = req.normalized()
	key := routeKey(req.Country, req.Currency, req.Method)
	for _, route := range c.Routes {
		route = route.normalized()
		if route.key() == key {
			return route, true
		}
	}
	return Route{}, false
}

func (r Route) normalized() Route {
	out := r
	out.Country = normalizeCountry(out.Country)
	out.Currency = normalizeCurrency(out.Currency)
	out.Method = normalizeMethod(out.Method)
	out.Providers = normalizeProviderIDs(out.Providers)
	return out
}

func (r Route) key() string {
	return routeKey(r.Country, r.Currency, r.Method)
}

func routeKey(country, currency string, method provider.PaymentMethod) string {
	return normalizeCountry(country) + "_" + normalizeCurrency(currency) + "_" + strings.ToUpper(string(normalizeMethod(method)))
}

func parseRouteEnv(key, value string) (Route, bool) {
	suffix := strings.TrimPrefix(key, envRoutePrefix)
	parts := strings.Split(suffix, "_")
	if len(parts) < 3 {
		return Route{}, false
	}

	providers := parseProviderList(value)
	if len(providers) == 0 {
		return Route{}, false
	}

	return Route{
		Country:   parts[0],
		Currency:  parts[1],
		Method:    provider.PaymentMethod(strings.ToLower(strings.Join(parts[2:], "_"))),
		Providers: providers,
	}.normalized(), true
}

func routesFromMap(items map[string]Route) []Route {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	routes := make([]Route, 0, len(items))
	for _, key := range keys {
		routes = append(routes, items[key])
	}
	return routes
}

func envMap(environ []string) map[string]string {
	out := make(map[string]string, len(environ))
	for _, item := range environ {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out
}

func parseProviderList(raw string) []provider.ID {
	parts := strings.Split(raw, ",")
	ids := make([]provider.ID, 0, len(parts))
	for _, part := range parts {
		id := normalizeProviderID(part)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return dedupeProviderIDs(ids)
}

func normalizeProviderIDs(ids []provider.ID) []provider.ID {
	out := make([]provider.ID, 0, len(ids))
	for _, id := range ids {
		id = normalizeProviderID(string(id))
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	return dedupeProviderIDs(out)
}

func dedupeProviderIDs(ids []provider.ID) []provider.ID {
	seen := map[provider.ID]struct{}{}
	out := make([]provider.ID, 0, len(ids))
	for _, id := range ids {
		id = normalizeProviderID(string(id))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeProviderID(raw string) provider.ID {
	return provider.ID(strings.ToLower(strings.TrimSpace(raw)))
}

func normalizeCountry(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func normalizeCurrency(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func normalizeMethod(method provider.PaymentMethod) provider.PaymentMethod {
	return provider.PaymentMethod(strings.ToLower(strings.TrimSpace(string(method))))
}

func parseBoolDefault(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
