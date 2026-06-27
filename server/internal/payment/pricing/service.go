package pricing

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MethodMobileMoney = "mobile_money"
	FeeModePassThrough = "pass_through"
)

var (
	ErrInvalidQuoteRequest = errors.New("invalid payment quote request")
	ErrNoFeeRule           = errors.New("no payment fee rule matched")
)

type Request struct {
	BaseAmount int64
	Country    string
	Currency   string
	Method     string
	Operator   string
}

type Quote struct {
	Country         string `json:"country"`
	Currency        string `json:"currency"`
	Method          string `json:"method"`
	Operator        string `json:"operator"`
	BaseAmount      int64  `json:"base_amount"`
	ProcessingFee   int64  `json:"processing_fee"`
	PayableAmount   int64  `json:"payable_amount"`
	FeeRateBps      int64  `json:"fee_rate_bps"`
	FeeFixedAmount  int64  `json:"fee_fixed_amount"`
	FeeMode         string `json:"fee_mode"`
}

type Rule struct {
	Country        string
	Currency       string
	Method         string
	Operator       string
	PercentBps     int64
	FixedMinor     int64
	Mode           string
}

type Service struct {
	rules []Rule
}

func NewService(rules []Rule) *Service {
	if len(rules) == 0 {
		rules = DefaultRules()
	}
	return &Service{rules: normalizeRules(rules)}
}

func NewDefaultService() *Service {
	return NewService(DefaultRules())
}

func DefaultRules() []Rule {
	return []Rule{
		{Country: "GH", Currency: "GHS", Method: MethodMobileMoney, Operator: "mtn", PercentBps: 200, Mode: FeeModePassThrough},
		{Country: "GH", Currency: "GHS", Method: MethodMobileMoney, Operator: "telecel", PercentBps: 200, Mode: FeeModePassThrough},
		{Country: "GH", Currency: "GHS", Method: MethodMobileMoney, Operator: "at", PercentBps: 200, Mode: FeeModePassThrough},
	}
}

func (s *Service) Quote(req Request) (*Quote, error) {
	req = normalizeRequest(req)
	if req.BaseAmount <= 0 {
		return nil, fmt.Errorf("%w: base_amount must be greater than zero", ErrInvalidQuoteRequest)
	}
	if req.Country == "" || req.Currency == "" || req.Method == "" || req.Operator == "" {
		return nil, fmt.Errorf("%w: country, currency, method, and operator are required", ErrInvalidQuoteRequest)
	}

	rule, ok := s.matchRule(req)
	if !ok {
		return nil, fmt.Errorf("%w for %s/%s/%s/%s", ErrNoFeeRule, req.Country, req.Currency, req.Method, req.Operator)
	}

	payableAmount, processingFee, err := calculateAmounts(req.BaseAmount, rule)
	if err != nil {
		return nil, err
	}

	return &Quote{
		Country:        req.Country,
		Currency:       req.Currency,
		Method:         req.Method,
		Operator:       req.Operator,
		BaseAmount:     req.BaseAmount,
		ProcessingFee:  processingFee,
		PayableAmount:  payableAmount,
		FeeRateBps:     rule.PercentBps,
		FeeFixedAmount: rule.FixedMinor,
		FeeMode:        rule.Mode,
	}, nil
}

func (s *Service) matchRule(req Request) (Rule, bool) {
	for _, rule := range s.rules {
		if rule.Country == req.Country && rule.Currency == req.Currency && rule.Method == req.Method && rule.Operator == req.Operator {
			return rule, true
		}
	}
	return Rule{}, false
}

func calculateAmounts(baseAmount int64, rule Rule) (payableAmount int64, processingFee int64, err error) {
	if rule.PercentBps < 0 || rule.PercentBps >= 10000 {
		return 0, 0, fmt.Errorf("%w: percent bps must be between 0 and 9999", ErrInvalidQuoteRequest)
	}
	if rule.FixedMinor < 0 {
		return 0, 0, fmt.Errorf("%w: fixed fee must be zero or greater", ErrInvalidQuoteRequest)
	}

	mode := strings.ToLower(strings.TrimSpace(rule.Mode))
	if mode == "" {
		mode = FeeModePassThrough
	}

	switch mode {
	case FeeModePassThrough:
		payableAmount = ceilDiv((baseAmount+rule.FixedMinor)*10000, 10000-rule.PercentBps)
		processingFee = payableAmount - baseAmount
		return payableAmount, processingFee, nil
	default:
		return 0, 0, fmt.Errorf("%w: unsupported fee mode %q", ErrInvalidQuoteRequest, rule.Mode)
	}
}

func ceilDiv(numerator, denominator int64) int64 {
	if denominator <= 0 {
		return 0
	}
	if numerator <= 0 {
		return 0
	}
	return (numerator + denominator - 1) / denominator
}

func normalizeRules(rules []Rule) []Rule {
	out := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		rule.Country = strings.ToUpper(strings.TrimSpace(rule.Country))
		rule.Currency = strings.ToUpper(strings.TrimSpace(rule.Currency))
		rule.Method = strings.ToLower(strings.TrimSpace(rule.Method))
		rule.Operator = normalizeOperator(rule.Operator)
		rule.Mode = strings.ToLower(strings.TrimSpace(rule.Mode))
		if rule.Mode == "" {
			rule.Mode = FeeModePassThrough
		}
		out = append(out, rule)
	}
	return out
}

func normalizeRequest(req Request) Request {
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	req.Operator = normalizeOperator(req.Operator)
	return req
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
	default:
		return operator
	}
}
