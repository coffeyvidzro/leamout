package markets

import "strings"

type Country struct {
	Code        string
	Name        string
	CallingCode string
	Currency    string
}

var countries = map[string]Country{
	"BEN": {Code: "BEN", Name: "Benin", CallingCode: "229", Currency: "XOF"},
	"BFA": {Code: "BFA", Name: "Burkina Faso", CallingCode: "226", Currency: "XOF"},
	"CMR": {Code: "CMR", Name: "Cameroon", CallingCode: "237", Currency: "XAF"},
	"GHA": {Code: "GHA", Name: "Ghana", CallingCode: "233", Currency: "GHS"},
	"CIV": {Code: "CIV", Name: "Ivory Coast", CallingCode: "225", Currency: "XOF"},
	"NGA": {Code: "NGA", Name: "Nigeria", CallingCode: "234", Currency: "NGN"},
	"SEN": {Code: "SEN", Name: "Senegal", CallingCode: "221", Currency: "XOF"},
	"SLE": {Code: "SLE", Name: "Sierra Leone", CallingCode: "232", Currency: "SLE"},
}

var countryAliases = map[string]string{
	"BEN":   "BEN",
	"BJ":    "BEN",
	"BENIN": "BEN",

	"BFA":          "BFA",
	"BF":           "BFA",
	"BURKINA FASO": "BFA",

	"CMR":      "CMR",
	"CM":       "CMR",
	"CAMEROON": "CMR",

	"GHA":   "GHA",
	"GH":    "GHA",
	"GHANA": "GHA",

	"CIV":           "CIV",
	"CI":            "CIV",
	"IVORY COAST":   "CIV",
	"COTE DIVOIRE":  "CIV",
	"CÔTE DIVOIRE":  "CIV",
	"COTE D'IVOIRE": "CIV",
	"CÔTE D'IVOIRE": "CIV",

	"NGA":     "NGA",
	"NG":      "NGA",
	"NIGERIA": "NGA",

	"SEN":     "SEN",
	"SN":      "SEN",
	"SENEGAL": "SEN",

	"SLE":          "SLE",
	"SL":           "SLE",
	"SIERRA LEONE": "SLE",
}

func NormalizeCountry(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.Join(strings.Fields(value), " ")

	code, ok := countryAliases[value]
	if !ok {
		return ""
	}

	return code
}

func CountryByCode(code string) (Country, bool) {
	code = NormalizeCountry(code)
	if code == "" {
		return Country{}, false
	}

	country, ok := countries[code]
	return country, ok
}

func CallingCode(country string) string {
	spec, ok := CountryByCode(country)
	if !ok {
		return ""
	}

	return spec.CallingCode
}

func DefaultCurrency(country string) string {
	spec, ok := CountryByCode(country)
	if !ok {
		return ""
	}

	return spec.Currency
}
