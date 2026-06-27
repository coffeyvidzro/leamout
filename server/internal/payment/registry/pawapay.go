package registry

import (
	"errors"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/pricing"
)

const (
	ProviderPawaPay      = "pawapay"
	MethodMobileMoney    = "mobile_money"
	FeeModePassThrough   = "pass_through"
	DecimalModeNone      = "NONE"
	DecimalModeTwoPlaces = "TWO_PLACES"
)

var (
	ErrMarketRuleNotFound   = errors.New("pawapay market rule not found")
	ErrFeeRuleNotConfigured = errors.New("pawapay fee rule not configured")
)

type PawaPayMarketRule struct {
	Country             string
	CountryAlpha3       string
	CountryName         string
	PhonePrefix         string
	Currency            string
	Method              string
	Operator            string
	OperatorDisplayName string
	ProviderCode        string
	MinAmountMinor      int64
	MaxAmountMinor      int64
	DecimalMode         string
	CollectionFeeBps    int64
	FeeMode             string
	FeeConfigured       bool
}

func PawaPayMVPRules() []PawaPayMarketRule {
	out := make([]PawaPayMarketRule, len(pawaPayMVPRules))
	copy(out, pawaPayMVPRules)
	return out
}

func FindPawaPayMVPRule(country, currency, operator string) (PawaPayMarketRule, bool) {
	country = normalizeCountry(country)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	operator = normalizeOperator(operator)
	for _, rule := range pawaPayMVPRules {
		if (rule.Country == country || rule.CountryAlpha3 == country) && rule.Currency == currency && rule.Operator == operator {
			return rule, true
		}
	}
	return PawaPayMarketRule{}, false
}

func FindPawaPayMVPRuleByProviderCode(providerCode string) (PawaPayMarketRule, bool) {
	providerCode = strings.ToUpper(strings.TrimSpace(providerCode))
	for _, rule := range pawaPayMVPRules {
		if rule.ProviderCode == providerCode {
			return rule, true
		}
	}
	return PawaPayMarketRule{}, false
}

func PawaPayMVPFeeRules() []pricing.Rule {
	rules := make([]pricing.Rule, 0, len(pawaPayMVPRules))
	for _, rule := range pawaPayMVPRules {
		if !rule.FeeConfigured {
			continue
		}
		rules = append(rules, rule.PricingRule())
	}
	return rules
}

func (r PawaPayMarketRule) PricingRule() pricing.Rule {
	return pricing.Rule{Country: r.Country, Currency: r.Currency, Method: r.Method, Operator: r.Operator, PercentBps: r.CollectionFeeBps, Mode: r.FeeMode}
}

func (r PawaPayMarketRule) ValidateAmount(amountMinor int64) bool {
	return amountMinor >= r.MinAmountMinor && amountMinor <= r.MaxAmountMinor
}

var pawaPayMVPRules = []PawaPayMarketRule{
	{Country: "BJ", CountryAlpha3: "BEN", CountryName: "Benin", PhonePrefix: "229", Currency: "XOF", Method: MethodMobileMoney, Operator: "moov", OperatorDisplayName: "Moov", ProviderCode: "MOOV_BEN", MinAmountMinor: 100, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 220, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "BJ", CountryAlpha3: "BEN", CountryName: "Benin", PhonePrefix: "229", Currency: "XOF", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_BEN", MinAmountMinor: 1, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 220, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "BF", CountryAlpha3: "BFA", CountryName: "Burkina Faso", PhonePrefix: "226", Currency: "XOF", Method: MethodMobileMoney, Operator: "moov", OperatorDisplayName: "Moov", ProviderCode: "MOOV_BFA", MinAmountMinor: 100, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 300, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CI", CountryAlpha3: "CIV", CountryName: "Ivory Coast", PhonePrefix: "225", Currency: "XOF", Method: MethodMobileMoney, Operator: "moov", OperatorDisplayName: "Moov", ProviderCode: "MOOV_CIV", MinAmountMinor: 5, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CI", CountryAlpha3: "CIV", CountryName: "Ivory Coast", PhonePrefix: "225", Currency: "XOF", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_CIV", MinAmountMinor: 1, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 180, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CI", CountryAlpha3: "CIV", CountryName: "Ivory Coast", PhonePrefix: "225", Currency: "XOF", Method: MethodMobileMoney, Operator: "orange", OperatorDisplayName: "Orange", ProviderCode: "ORANGE_CIV", MinAmountMinor: 1, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 250, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CM", CountryAlpha3: "CMR", CountryName: "Cameroon", PhonePrefix: "237", Currency: "XAF", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_CMR", MinAmountMinor: 1, MaxAmountMinor: 1000000, DecimalMode: "NONE", CollectionFeeBps: 175, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CM", CountryAlpha3: "CMR", CountryName: "Cameroon", PhonePrefix: "237", Currency: "XAF", Method: MethodMobileMoney, Operator: "orange", OperatorDisplayName: "Orange", ProviderCode: "ORANGE_CMR", MinAmountMinor: 1, MaxAmountMinor: 500000, DecimalMode: "NONE", CollectionFeeBps: 177, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CD", CountryAlpha3: "COD", CountryName: "Democratic Republic of the Congo", PhonePrefix: "243", Currency: "CDF", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_COD", MinAmountMinor: 10000, MaxAmountMinor: 625000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 300, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CD", CountryAlpha3: "COD", CountryName: "Democratic Republic of the Congo", PhonePrefix: "243", Currency: "CDF", Method: MethodMobileMoney, Operator: "orange", OperatorDisplayName: "Orange", ProviderCode: "ORANGE_COD", MinAmountMinor: 1000, MaxAmountMinor: 100000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 300, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CD", CountryAlpha3: "COD", CountryName: "Democratic Republic of the Congo", PhonePrefix: "243", Currency: "CDF", Method: MethodMobileMoney, Operator: "vodacom", OperatorDisplayName: "Vodacom", ProviderCode: "VODACOM_MPESA_COD", MinAmountMinor: 500, MaxAmountMinor: 1000000, DecimalMode: "NONE", CollectionFeeBps: 250, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CG", CountryAlpha3: "COG", CountryName: "Congo", PhonePrefix: "242", Currency: "XAF", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_COG", MinAmountMinor: 10, MaxAmountMinor: 1500000, DecimalMode: "NONE", CollectionFeeBps: 400, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "CG", CountryAlpha3: "COG", CountryName: "Congo", PhonePrefix: "242", Currency: "XAF", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_COG", MinAmountMinor: 1, MaxAmountMinor: 1000000, DecimalMode: "NONE", CollectionFeeBps: 400, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "GA", CountryAlpha3: "GAB", CountryName: "Gabon", PhonePrefix: "241", Currency: "XAF", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_GAB", MinAmountMinor: 10000, MaxAmountMinor: 50000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "GH", CountryAlpha3: "GHA", CountryName: "Ghana", PhonePrefix: "233", Currency: "GHS", Method: MethodMobileMoney, Operator: "at", OperatorDisplayName: "AT", ProviderCode: "AIRTELTIGO_GHA", MinAmountMinor: 100, MaxAmountMinor: 1000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "GH", CountryAlpha3: "GHA", CountryName: "Ghana", PhonePrefix: "233", Currency: "GHS", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_GHA", MinAmountMinor: 100, MaxAmountMinor: 1500000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "GH", CountryAlpha3: "GHA", CountryName: "Ghana", PhonePrefix: "233", Currency: "GHS", Method: MethodMobileMoney, Operator: "telecel", OperatorDisplayName: "Telecel", ProviderCode: "VODAFONE_GHA", MinAmountMinor: 100, MaxAmountMinor: 1500000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "MW", CountryAlpha3: "MWI", CountryName: "Malawi", PhonePrefix: "265", Currency: "MWK", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_MWI", MinAmountMinor: 5000, MaxAmountMinor: 75000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 333, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "MW", CountryAlpha3: "MWI", CountryName: "Malawi", PhonePrefix: "265", Currency: "MWK", Method: MethodMobileMoney, Operator: "tnm", OperatorDisplayName: "TNM", ProviderCode: "TNM_MWI", MinAmountMinor: 5000, MaxAmountMinor: 75000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 333, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "RW", CountryAlpha3: "RWA", CountryName: "Rwanda", PhonePrefix: "250", Currency: "RWF", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_RWA", MinAmountMinor: 100, MaxAmountMinor: 1500000, DecimalMode: "NONE", CollectionFeeBps: 250, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "RW", CountryAlpha3: "RWA", CountryName: "Rwanda", PhonePrefix: "250", Currency: "RWF", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_RWA", MinAmountMinor: 5, MaxAmountMinor: 2000000, DecimalMode: "NONE", CollectionFeeBps: 310, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "SN", CountryAlpha3: "SEN", CountryName: "Senegal", PhonePrefix: "221", Currency: "XOF", Method: MethodMobileMoney, Operator: "free", OperatorDisplayName: "Free", ProviderCode: "FREE_SEN", MinAmountMinor: 5, MaxAmountMinor: 100000000, DecimalMode: "NONE", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "SN", CountryAlpha3: "SEN", CountryName: "Senegal", PhonePrefix: "221", Currency: "XOF", Method: MethodMobileMoney, Operator: "orange", OperatorDisplayName: "Orange", ProviderCode: "ORANGE_SEN", MinAmountMinor: 2, MaxAmountMinor: 100000000, DecimalMode: "NONE", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "SL", CountryAlpha3: "SLE", CountryName: "Sierra Leone", PhonePrefix: "232", Currency: "SLE", Method: MethodMobileMoney, Operator: "orange", OperatorDisplayName: "Orange", ProviderCode: "ORANGE_SLE", MinAmountMinor: 100, MaxAmountMinor: 1500000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 330, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "TZ", CountryAlpha3: "TZA", CountryName: "Tanzania", PhonePrefix: "255", Currency: "TZS", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_TZA", MinAmountMinor: 10000, MaxAmountMinor: 300000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 218, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "TZ", CountryAlpha3: "TZA", CountryName: "Tanzania", PhonePrefix: "255", Currency: "TZS", Method: MethodMobileMoney, Operator: "halotel", OperatorDisplayName: "Halotel", ProviderCode: "HALOTEL_TZA", MinAmountMinor: 100, MaxAmountMinor: 5000000, DecimalMode: "NONE", CollectionFeeBps: 200, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "TZ", CountryAlpha3: "TZA", CountryName: "Tanzania", PhonePrefix: "255", Currency: "TZS", Method: MethodMobileMoney, Operator: "tigo", OperatorDisplayName: "Tigo", ProviderCode: "TIGO_TZA", MinAmountMinor: 100, MaxAmountMinor: 5000000, DecimalMode: "NONE", CollectionFeeBps: 100, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "UG", CountryAlpha3: "UGA", CountryName: "Uganda", PhonePrefix: "256", Currency: "UGX", Method: MethodMobileMoney, Operator: "airtel", OperatorDisplayName: "Airtel", ProviderCode: "AIRTEL_OAPI_UGA", MinAmountMinor: 500, MaxAmountMinor: 5000000, DecimalMode: "NONE", CollectionFeeBps: 250, FeeMode: FeeModePassThrough, FeeConfigured: true},
	{Country: "UG", CountryAlpha3: "UGA", CountryName: "Uganda", PhonePrefix: "256", Currency: "UGX", Method: MethodMobileMoney, Operator: "mtn", OperatorDisplayName: "MTN", ProviderCode: "MTN_MOMO_UGA", MinAmountMinor: 50000, MaxAmountMinor: 500000000, DecimalMode: "TWO_PLACES", CollectionFeeBps: 300, FeeMode: FeeModePassThrough, FeeConfigured: true},
}

func normalizeCountry(country string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	switch country {
	case "BEN":
		return "BJ"
	case "BFA":
		return "BF"
	case "CIV":
		return "CI"
	case "CMR":
		return "CM"
	case "COD":
		return "CD"
	case "COG":
		return "CG"
	case "GAB":
		return "GA"
	case "GHA":
		return "GH"
	case "MWI":
		return "MW"
	case "RWA":
		return "RW"
	case "SEN":
		return "SN"
	case "SLE":
		return "SL"
	case "TZA":
		return "TZ"
	case "UGA":
		return "UG"
	default:
		return country
	}
}

func normalizeOperator(operator string) string {
	operator = strings.ToLower(strings.TrimSpace(operator))
	operator = strings.ReplaceAll(operator, " ", "_")
	operator = strings.ReplaceAll(operator, "-", "_")
	switch operator {
	case "mtn", "mtn_momo", "mtn_mobile_money":
		return "mtn"
	case "telecel", "telecel_cash", "vodafone", "vodafone_cash":
		return "telecel"
	case "at", "airteltigo", "airtel_tigo", "at_money":
		return "at"
	case "safaricom", "mpesa", "m_pesa":
		return "mpesa"
	case "yas":
		return "tigo"
	default:
		return operator
	}
}
