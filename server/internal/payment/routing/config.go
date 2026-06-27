package routing

import (
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type Route struct {
	Country  string
	Network  string
	Currency string

	Provider payment.ProviderName
	Operator string

	Priority int
	Enabled  bool
}

type Config struct {
	routes map[string][]Route
}

func NewConfig(routes []Route) *Config {
	cfg := &Config{
		routes: make(map[string][]Route),
	}

	for _, route := range routes {
		cfg.AddRoute(route)
	}

	return cfg
}

func NewDefaultConfig() *Config {
	return NewConfig([]Route{
		// Benin - XOF
		// pawaPay supported. Tola not added because Benin is not in the Tola active country list.
		{
			Country:  "BEN",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MOOV_BEN",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "BEN",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_BEN",
			Priority: 1,
			Enabled:  true,
		},

		// Burkina Faso - XOF
		{
			Country:  "BFA",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MOOV_BFA",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "BFA",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "BURKINA_FASO.MOOV",
			Priority: 2,
			Enabled:  true,
		},

		// Cameroon - XAF
		{
			Country:  "CMR",
			Network:  "MTN",
			Currency: "XAF",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_CMR",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "MTN",
			Currency: "XAF",
			Provider: payment.ProviderTola,
			Operator: "CAMEROON.MTN",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "ORANGE",
			Currency: "XAF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_CMR",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "ORANGE",
			Currency: "XAF",
			Provider: payment.ProviderTola,
			Operator: "CAMEROON.ORANGE",
			Priority: 2,
			Enabled:  true,
		},

		// Ghana - GHS
		{
			Country:  "GHA",
			Network:  "AIRTELTIGO",
			Currency: "GHS",
			Provider: payment.ProviderPawaPay,
			Operator: "AIRTELTIGO_GHA",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "AIRTELTIGO",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.AIRTELTIGO",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "MTN",
			Currency: "GHS",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_GHA",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "MTN",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.MTN",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "TELECEL",
			Currency: "GHS",
			Provider: payment.ProviderPawaPay,
			Operator: "VODAFONE_GHA",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "TELECEL",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.TELECEL",
			Priority: 2,
			Enabled:  true,
		},

		// Ivory Coast - XOF
		{
			Country:  "CIV",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MOOV_CIV",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "IVORY_COAST.MOOV",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_CIV",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "IVORY_COAST.MTN",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_CIV",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "IVORY_COAST.ORANGE",
			Priority: 2,
			Enabled:  true,
		},

		// Senegal - XOF
		{
			Country:  "SEN",
			Network:  "FREE",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "FREE_SEN",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "SEN",
			Network:  "FREE",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "SENEGAL.FREE",
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "SEN",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_SEN",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "SEN",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "SENEGAL.ORANGE",
			Priority: 2,
			Enabled:  true,
		},

		// Sierra Leone - SLE
		{
			Country:  "SLE",
			Network:  "ORANGE",
			Currency: "SLE",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_SLE",
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "SLE",
			Network:  "ORANGE",
			Currency: "SLE",
			Provider: payment.ProviderTola,
			Operator: "SIERRA_LEONE.ORANGE",
			Priority: 2,
			Enabled:  true,
		},

		// Nigeria intentionally not added yet.
		// pawaPay support was listed, but the uploaded pawaPay provider dump does not include Nigeria operator codes.
		// Tola's public active-country list does not include Nigeria.
	})
}

func (c *Config) AddRoute(route Route) {
	route.Country = normalize(route.Country)
	route.Network = normalize(route.Network)
	route.Currency = normalize(route.Currency)
	route.Operator = strings.TrimSpace(route.Operator)

	if route.Priority <= 0 {
		route.Priority = 100
	}

	key := routeKey(route.Country, route.Network, route.Currency)
	c.routes[key] = append(c.routes[key], route)
}

func (c *Config) Lookup(country, network, currency string) ([]Route, error) {
	if c == nil {
		return nil, fmt.Errorf("missing routing config")
	}

	country = normalize(country)
	network = normalize(network)
	currency = normalize(currency)

	key := routeKey(country, network, currency)

	routes, ok := c.routes[key]
	if !ok || len(routes) == 0 {
		return nil, fmt.Errorf(
			"unsupported payment route: country=%s network=%s currency=%s",
			country,
			network,
			currency,
		)
	}

	return routes, nil
}

func routeKey(country, network, currency string) string {
	return normalize(country) + ":" + normalize(network) + ":" + normalize(currency)
}

func normalize(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
