package paymentmethod

import "strings"

const methodMobileMoney = "mobile_money"

type Service struct {
	countries []Country
}

func NewService() *Service {
	return &Service{countries: defaultCountries()}
}

func (s *Service) List(params ListParams) CatalogResponse {
	params = normalizeParams(params)

	countries := make([]Country, 0, len(s.countries))
	for _, country := range s.countries {
		if params.Country != "" && country.Code != params.Country {
			continue
		}
		if params.Currency != "" && country.DefaultCurrency != params.Currency {
			continue
		}

		country.SupportedMethods = filterMethods(country.SupportedMethods, params.Method)
		if len(country.SupportedMethods) == 0 {
			continue
		}

		countries = append(countries, country)
	}

	return CatalogResponse{Countries: countries}
}

func defaultCountries() []Country {
	return []Country{
		{
			Code:            "GH",
			Name:            "Ghana",
			CallingCode:     "+233",
			DefaultCurrency: "GHS",
			SupportedMethods: []Method{
				{
					Type: methodMobileMoney,
					Operators: []Operator{
						{Code: "mtn", DisplayName: "MTN MoMo"},
						{Code: "telecel", DisplayName: "Telecel Cash"},
						{Code: "at", DisplayName: "AT Money"},
					},
				},
			},
		},
	}
}

func filterMethods(methods []Method, method string) []Method {
	if method == "" {
		return methods
	}

	out := make([]Method, 0, len(methods))
	for _, item := range methods {
		if item.Type == method {
			out = append(out, item)
		}
	}
	return out
}

func normalizeParams(params ListParams) ListParams {
	params.Country = strings.ToUpper(strings.TrimSpace(params.Country))
	params.Currency = strings.ToUpper(strings.TrimSpace(params.Currency))
	params.Method = strings.ToLower(strings.TrimSpace(params.Method))
	return params
}
