package paymentmethod

import "strings"

const (
	methodMobileMoney = "mobile_money"
	statusAvailable   = "available"
	statusComingSoon  = "coming_soon"
)

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
		if params.Currency != "" && !containsString(country.Currencies, params.Currency) {
			continue
		}
		if params.Status != "" && country.Status != params.Status {
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
		country("GH", "Ghana", "+233", "GHS", []string{"GHS"}, statusAvailable, operators(
			op("mtn", "MTN MoMo"),
			op("telecel", "Telecel Cash"),
			op("at", "AT Money"),
		)),
		country("BJ", "Benin", "+229", "XOF", []string{"XOF"}, statusComingSoon, operators(
			op("moov", "Moov"),
			op("mtn", "MTN"),
		)),
		country("BF", "Burkina Faso", "+226", "XOF", []string{"XOF"}, statusComingSoon, operators(
			op("moov", "Moov"),
		)),
		country("CI", "Ivory Coast", "+225", "XOF", []string{"XOF"}, statusComingSoon, operators(
			op("moov", "Moov"),
			op("mtn", "MTN"),
			op("orange", "Orange"),
		)),
		country("CM", "Cameroon", "+237", "XAF", []string{"XAF"}, statusComingSoon, operators(
			op("mtn", "MTN"),
			op("orange", "Orange"),
		)),
		country("CD", "Democratic Republic of the Congo", "+243", "CDF", []string{"CDF", "USD"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("orange", "Orange"),
			op("vodacom", "Vodacom"),
		)),
		country("CG", "Congo", "+242", "XAF", []string{"XAF"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("mtn", "MTN"),
		)),
		country("GA", "Gabon", "+241", "XAF", []string{"XAF"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
		)),
		country("KE", "Kenya", "+254", "KES", []string{"KES"}, statusComingSoon, operators(
			op("mpesa", "Safaricom M-Pesa"),
		)),
		country("MZ", "Mozambique", "+258", "MZN", []string{"MZN"}, statusComingSoon, operators(
			op("vodacom", "Vodacom"),
		)),
		country("MW", "Malawi", "+265", "MWK", []string{"MWK"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("tnm", "TNM"),
		)),
		country("RW", "Rwanda", "+250", "RWF", []string{"RWF"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("mtn", "MTN"),
		)),
		country("SN", "Senegal", "+221", "XOF", []string{"XOF"}, statusComingSoon, operators(
			op("free", "Free"),
			op("orange", "Orange"),
		)),
		country("SL", "Sierra Leone", "+232", "SLE", []string{"SLE"}, statusComingSoon, operators(
			op("orange", "Orange"),
		)),
		country("TZ", "Tanzania", "+255", "TZS", []string{"TZS"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("halotel", "Halotel"),
			op("tigo", "Tigo"),
		)),
		country("UG", "Uganda", "+256", "UGX", []string{"UGX"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("mtn", "MTN"),
		)),
		country("ZM", "Zambia", "+260", "ZMW", []string{"ZMW"}, statusComingSoon, operators(
			op("airtel", "Airtel"),
			op("mtn", "MTN"),
			op("zamtel", "Zamtel"),
		)),
	}
}

func country(code, name, callingCode, defaultCurrency string, currencies []string, status string, operators []Operator) Country {
	return Country{
		Code:            code,
		Name:            name,
		CallingCode:     callingCode,
		DefaultCurrency: defaultCurrency,
		Currencies:      currencies,
		Status:          status,
		SupportedMethods: []Method{
			{
				Type:      methodMobileMoney,
				Operators: operators,
			},
		},
	}
}

func operators(items ...Operator) []Operator {
	return items
}

func op(code, displayName string) Operator {
	return Operator{Code: code, DisplayName: displayName}
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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func normalizeParams(params ListParams) ListParams {
	params.Country = strings.ToUpper(strings.TrimSpace(params.Country))
	params.Currency = strings.ToUpper(strings.TrimSpace(params.Currency))
	params.Method = strings.ToLower(strings.TrimSpace(params.Method))
	params.Status = strings.ToLower(strings.TrimSpace(params.Status))
	return params
}
