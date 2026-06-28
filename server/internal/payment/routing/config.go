package routing

import (
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

type RouteFees struct {
	MMOFeeBps      int64 `json:"mmo_fee_bps"`      // actual mobile money operator fee
	ProviderFeeBps int64 `json:"provider_fee_bps"` // provider fee + Leamout cushion
}

func (f RouteFees) TotalFeeBps() int64 {
	return f.MMOFeeBps + f.ProviderFeeBps
}

type Route struct {
	Country  string
	Network  string
	Currency string

	Provider payment.ProviderName
	Operator string

	Fees RouteFees

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
			Fees: RouteFees{
				MMOFeeBps:      120, // 1.2%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "BEN",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_BEN",
			Fees: RouteFees{
				MMOFeeBps:      120, // 1.2%
				ProviderFeeBps: 150, // 1.5%
			},
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
			Fees: RouteFees{
				MMOFeeBps:      200, // 2.0% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "BFA",
			Network:  "MOOV",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "BURKINA_FASO.MOOV",
			Fees: RouteFees{
				MMOFeeBps:      200, // 2.0% MMO
				ProviderFeeBps: 150, // Tola/provider + Leamout cushion
			},
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "BFA",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_BFA",
			Fees: RouteFees{
				MMOFeeBps:      230, // 2.3% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "BFA",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "BURKINA_FASO.ORANGE",
			Fees: RouteFees{
				MMOFeeBps:      230, // 2.3% MMO
				ProviderFeeBps: 150, // Tola/provider + Leamout cushion
			},
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
			Fees: RouteFees{
				MMOFeeBps:      75,  // 0.75% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "MTN",
			Currency: "XAF",
			Provider: payment.ProviderTola,
			Operator: "CAMEROON.MTN",
			Fees: RouteFees{
				MMOFeeBps:      75,  // 0.75% MMO
				ProviderFeeBps: 150, // Tola/provider + Leamout cushion
			},
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "ORANGE",
			Currency: "XAF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_CMR",
			Fees: RouteFees{
				MMOFeeBps:      77,  // 0.77% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CMR",
			Network:  "ORANGE",
			Currency: "XAF",
			Provider: payment.ProviderTola,
			Operator: "CAMEROON.ORANGE",
			Fees: RouteFees{
				MMOFeeBps:      77,  // 0.77% MMO
				ProviderFeeBps: 150, // Tola/provider + Leamout cushion
			},
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
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "AIRTELTIGO",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.AIRTELTIGO",
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "MTN",
			Currency: "GHS",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_GHA",
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "MTN",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.MTN",
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "TELECEL",
			Currency: "GHS",
			Provider: payment.ProviderPawaPay,
			Operator: "VODAFONE_GHA",
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "GHA",
			Network:  "TELECEL",
			Currency: "GHS",
			Provider: payment.ProviderTola,
			Operator: "GHANA.TELECEL",
			Fees: RouteFees{
				MMOFeeBps:      100, // 1.0%
				ProviderFeeBps: 150, // 1.5%
			},
			Priority: 2,
			Enabled:  true,
		},

		// Ivory Coast - XOF
		{
			Country:  "CIV",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "MTN_MOMO_CIV",
			Fees: RouteFees{
				MMOFeeBps:      80,  // 0.80% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "MTN",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "IVORY_COAST.MTN",
			Fees: RouteFees{
				MMOFeeBps:      80,  // 0.80% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 2,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderPawaPay,
			Operator: "ORANGE_CIV",
			Fees: RouteFees{
				MMOFeeBps:      150, // 1.5% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "CIV",
			Network:  "ORANGE",
			Currency: "XOF",
			Provider: payment.ProviderTola,
			Operator: "IVORY_COAST.ORANGE",
			Fees: RouteFees{
				MMOFeeBps:      150, // 1.5% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
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
			Fees: RouteFees{
				MMOFeeBps:      230, // 2.3% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
			Priority: 1,
			Enabled:  true,
		},
		{
			Country:  "SLE",
			Network:  "ORANGE",
			Currency: "SLE",
			Provider: payment.ProviderTola,
			Operator: "SIERRA_LEONE.ORANGE",
			Fees: RouteFees{
				MMOFeeBps:      230, // 2.3% MMO
				ProviderFeeBps: 150, // 1.0% pawaPay + 0.5% Leamout cushion
			},
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
