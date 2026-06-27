package routing

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

const (
	envAllowFallback      = "PAYMENT_ROUTING_ALLOW_FALLBACK"
	envStrictCapabilities = "PAYMENT_ROUTING_STRICT_CAPABILITIES"
)

type Config struct {
	EnabledProviders   []provider.ID `json:"enabled_providers"`
	DefaultProvider    provider.ID   `json:"default_provider,omitempty"`
	Routes             []Route       `json:"routes"`
	AllowFallback      bool          `json:"allow_fallback"`
	StrictCapabilities bool          `json:"strict_capabilities"`
}

type Route struct {
	Country   string                 `json:"country"`
	Currency  string                 `json:"currency"`
	Method    provider.PaymentMethod `json:"method"`
	Providers []provider.ID          `json:"providers"`
}

func DefaultConfig() Config {
	return Config{
		EnabledProviders: []provider.ID{provider.ProviderPawaPay},
		DefaultProvider:  provider.ProviderPawaPay,
		Routes: []Route{
			{Country: "GH", Currency: "GHS", Method: provider.PaymentMethodMobileMoney, Providers: []provider.ID{provider.ProviderPawaPay}},
		},
		AllowFallback:      false,
		StrictCapabilities: true,
	}
}

func LoadConfigFromEnv() Config {
	return ConfigFromEnv(os.Environ())
}

func ConfigFromEnv(environ []string) Config {
	cfg := DefaultConfig()
	env := envMap(environ)

	// MVP is intentionally locked to PawaPay. Env can only tune behavior flags;
	// it cannot switch providers or introduce fallback routes yet.
	cfg.AllowFallback = false
	cfg.StrictCapabilities = parseBoolDefault(env[envStrictCapabilities], cfg.StrictCapabilities)
	if parseBoolDefault(env[envAllowFallback], false) {
		cfg.AllowFallback = false
	}

	return cfg.normalized()
}

func (c Config) normalized() Config {
	out := c
	out.EnabledProviders = []provider.ID{provider.ProviderPawaPay}
	out.DefaultProvider = provider.ProviderPawaPay
	out.AllowFallback = false

	out.Routes = make([]Route, 0, len(c.Routes))
	for _, route := range c.Routes {
		route = route.normalized()
		if route.Country == "" || route.Currency == "" || route.Method == "" {
			continue
		}
		route.Providers = []provider.ID{provider.ProviderPawaPay}
		out.Routes = append(out.Routes, route)
	}
	if len(out.Routes) == 0 {
		out.Routes = DefaultConfig().Routes
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
	if c.DefaultProvider != provider.ProviderPawaPay {
		return fmt.Errorf("default payment provider must be %q for MVP", provider.ProviderPawaPay)
	}
	for _, id := range c.EnabledProviders {
		if id != provider.ProviderPawaPay {
			return fmt.Errorf("payment provider %q is not allowed in MVP", id)
		}
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
		for _, id := range route.Providers {
			if id != provider.ProviderPawaPay {
				return fmt.Errorf("payment route %s uses non-MVP provider %q", route.key(), id)
			}
		}
	}
	return nil
}

func (c Config) IsProviderEnabled(id provider.ID) bool {
	return normalizeProviderID(string(id)) == provider.ProviderPawaPay
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
	out.Providers = []provider.ID{provider.ProviderPawaPay}
	return out
}

func (r Route) key() string {
	return routeKey(r.Country, r.Currency, r.Method)
}

func routeKey(country, currency string, method provider.PaymentMethod) string {
	return normalizeCountry(country) + "_" + normalizeCurrency(currency) + "_" + strings.ToUpper(string(normalizeMethod(method)))
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

func parseBoolDefault(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		return fallback
	}
}
