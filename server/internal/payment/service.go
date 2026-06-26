package payment

import (
	"context"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
	"github.com/cuffeyvidzro/leamout/internal/payment/routing"
)

// Router is the small part of routing.Service that payment orchestration needs.
// Keeping this as an interface makes the package easy to test.
type Router interface {
	ResolveProvider(ctx context.Context, req routing.RouteRequest) (provider.Provider, *routing.RouteResult, error)
	Provider(id provider.ID) (provider.Provider, bool)
}

// Service is the provider-neutral payment kernel for Leamout.
//
// It owns payment orchestration only:
//   - validate checkout/dunning payment requests
//   - choose a provider through routing
//   - map provider-neutral MoMo operator values to provider options
//   - call provider.InitiatePayment
//   - call provider.VerifyPayment
//   - reconcile webhook events by verifying payment status
//
// It must not import moolre, pawapay, checkout, dunning, or subscription.
type Service struct {
	router Router
	cfg    Config
	hooks  Hooks
}

func NewService(router Router, cfg Config, hooks Hooks) *Service {
	if hooks == nil {
		hooks = NoopHooks{}
	}

	// Preserve explicit false values when caller passes a zero Config by merging
	// with defaults only for fields that do not have meaningful false defaults.
	defaults := DefaultConfig()
	if !cfg.VerifyWebhookPayments {
		// For safety, default zero-value config to verification enabled. To disable,
		// use NewServiceWithoutWebhookVerification or set after construction.
		cfg.VerifyWebhookPayments = defaults.VerifyWebhookPayments
	}
	if !cfg.NormalizeCustomerPhone {
		cfg.NormalizeCustomerPhone = defaults.NormalizeCustomerPhone
	}

	return &Service{router: router, cfg: cfg, hooks: hooks}
}

func NewServiceWithDefaults(router Router, hooks Hooks) *Service {
	return NewService(router, DefaultConfig(), hooks)
}

// NewServiceWithoutWebhookVerification is useful for local tests only. In
// production, keep webhook verification/reconciliation enabled.
func NewServiceWithoutWebhookVerification(router Router, hooks Hooks) *Service {
	cfg := DefaultConfig()
	cfg.VerifyWebhookPayments = false
	return &Service{router: router, cfg: cfg, hooks: normalizeHooks(hooks)}
}

func (s *Service) InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*InitiatePaymentResult, error) {
	if s == nil || s.router == nil {
		return nil, ErrRouterUnavailable
	}

	req = normalizeInitiateRequest(req, s.cfg)
	if err := validateInitiateRequest(req); err != nil {
		return nil, err
	}

	selectedProvider, routeResult, err := s.router.ResolveProvider(ctx, routing.RouteRequest{
		Country:           req.Country,
		Currency:          req.Currency,
		Method:            req.Method,
		PreferredProvider: req.PreferredProvider,
		AmountMinor:       req.AmountMinor,
	})
	if err != nil {
		return nil, err
	}
	if selectedProvider == nil {
		return nil, ErrProviderUnavailable
	}

	providerReq := toProviderInitiateRequest(req)
	providerReq.ProviderOptions = enrichProviderOptions(
		selectedProvider.ID(),
		req.Country,
		req.Operator,
		providerReq.ProviderOptions,
	)

	providerResp, err := selectedProvider.InitiatePayment(ctx, providerReq)
	if err != nil {
		return nil, err
	}
	if providerResp == nil {
		return nil, fmt.Errorf("%w: provider returned nil initiate response", ErrProviderUnavailable)
	}

	result := toInitiateResult(selectedProvider, routeResult, providerResp)
	if result.ExternalRef == "" {
		result.ExternalRef = req.ExternalRef
	}
	if result.ProviderID == "" {
		result.ProviderID = selectedProvider.ID()
	}
	if result.ProviderName == "" {
		result.ProviderName = selectedProvider.Name()
	}

	if err := s.hooks.PaymentInitiated(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) VerifyPayment(ctx context.Context, req VerifyPaymentRequest) (*VerifyPaymentResult, error) {
	if s == nil || s.router == nil {
		return nil, ErrRouterUnavailable
	}

	req = normalizeVerifyRequest(req)
	if err := validateVerifyRequest(req); err != nil {
		return nil, err
	}

	paymentProvider, ok := s.router.Provider(req.ProviderID)
	if !ok || paymentProvider == nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, req.ProviderID)
	}

	if !paymentProvider.Capabilities().SupportsVerifyPayment {
		return nil, fmt.Errorf("%w: provider %s does not support payment verification", ErrVerificationFailed, req.ProviderID)
	}

	providerResp, err := paymentProvider.VerifyPayment(ctx, provider.VerifyPaymentRequest{
		ExternalRef:       req.ExternalRef,
		ProviderReference: req.ProviderReference,
	})
	if err != nil {
		return nil, err
	}
	if providerResp == nil {
		return nil, fmt.Errorf("%w: provider returned nil verification response", ErrVerificationFailed)
	}

	result := toVerifyResult(providerResp)
	if result.ProviderID == "" {
		result.ProviderID = req.ProviderID
	}
	if result.ExternalRef == "" {
		result.ExternalRef = req.ExternalRef
	}
	if result.ProviderReference == "" {
		result.ProviderReference = req.ProviderReference
	}

	if err := s.hooks.PaymentVerified(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProcessWebhookEvent implements the webhook.Processor shape without importing
// the webhook package. Webhook handlers pass normalized provider.WebhookEvent
// here; this service verifies/reconciles with the provider before app/domain code
// acts on it.
func (s *Service) ProcessWebhookEvent(ctx context.Context, event *provider.WebhookEvent) error {
	_, err := s.ProcessWebhook(ctx, event)
	return err
}

func (s *Service) ProcessWebhook(ctx context.Context, event *provider.WebhookEvent) (*ProcessedWebhookResult, error) {
	if event == nil {
		return nil, fmt.Errorf("%w: webhook event is nil", ErrInvalidRequest)
	}

	event.ProviderID = normalizeProviderID(event.ProviderID)
	if event.ProviderID == "" {
		return nil, fmt.Errorf("%w: webhook provider_id is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(event.ExternalRef) == "" && strings.TrimSpace(event.ProviderReference) == "" {
		return nil, fmt.Errorf("%w: webhook needs external_ref or provider_reference", ErrInvalidRequest)
	}

	result := &ProcessedWebhookResult{
		ProviderID:        event.ProviderID,
		EventType:         strings.TrimSpace(event.EventType),
		Status:            event.Status,
		ExternalRef:       strings.TrimSpace(event.ExternalRef),
		ProviderReference: strings.TrimSpace(event.ProviderReference),
		Verified:          event.Verified,
		Metadata:          cloneStringMap(event.Metadata),
	}

	if s.cfg.VerifyWebhookPayments {
		verification, err := s.VerifyPayment(ctx, VerifyPaymentRequest{
			ProviderID:        event.ProviderID,
			ExternalRef:       event.ExternalRef,
			ProviderReference: event.ProviderReference,
		})
		if err != nil {
			return nil, err
		}

		result.Verification = verification
		result.Status = verification.Status
		if result.ExternalRef == "" {
			result.ExternalRef = verification.ExternalRef
		}
		if result.ProviderReference == "" {
			result.ProviderReference = verification.ProviderReference
		}
	}

	if err := s.hooks.WebhookProcessed(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

func normalizeInitiateRequest(req InitiatePaymentRequest, cfg Config) InitiatePaymentRequest {
	req.UserID = strings.TrimSpace(req.UserID)
	req.ExternalRef = strings.TrimSpace(req.ExternalRef)
	req.Currency = normalizeCurrency(req.Currency)
	req.Country = normalizeCountry(req.Country)
	req.Method = normalizePaymentMethod(req.Method)
	if req.Method == "" {
		req.Method = provider.PaymentMethodMobileMoney
	}
	req.Operator = normalizeOperator(req.Operator)
	req.PreferredProvider = normalizeProviderID(req.PreferredProvider)
	req.Description = strings.TrimSpace(req.Description)
	req.CallbackURL = strings.TrimSpace(req.CallbackURL)
	req.ReturnURL = strings.TrimSpace(req.ReturnURL)
	req.Customer = normalizeCustomer(req.Customer, req.Country, cfg.NormalizeCustomerPhone)
	req.Metadata = cloneStringMap(req.Metadata)
	req.ProviderOptions = cloneAnyMap(req.ProviderOptions)
	return req
}

func validateInitiateRequest(req InitiatePaymentRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidRequest)
	}
	if req.ExternalRef == "" {
		return fmt.Errorf("%w: external_ref is required", ErrInvalidRequest)
	}
	if req.AmountMinor <= 0 {
		return fmt.Errorf("%w: amount_minor must be greater than zero", ErrInvalidRequest)
	}
	if req.Currency == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidRequest)
	}
	if req.Country == "" {
		return fmt.Errorf("%w: country is required", ErrInvalidRequest)
	}
	if req.Method == "" {
		return fmt.Errorf("%w: method is required", ErrInvalidRequest)
	}
	if req.Method == provider.PaymentMethodMobileMoney {
		if req.Customer.Phone == "" {
			return fmt.Errorf("%w: customer.phone is required", ErrInvalidRequest)
		}
		if req.Operator == "" && !hasProviderOperatorOptions(req.ProviderOptions) {
			return fmt.Errorf("%w: mobile money operator is required", ErrInvalidRequest)
		}
	}
	return nil
}

func normalizeVerifyRequest(req VerifyPaymentRequest) VerifyPaymentRequest {
	req.ProviderID = normalizeProviderID(req.ProviderID)
	req.ExternalRef = strings.TrimSpace(req.ExternalRef)
	req.ProviderReference = strings.TrimSpace(req.ProviderReference)
	return req
}

func validateVerifyRequest(req VerifyPaymentRequest) error {
	if req.ProviderID == "" {
		return fmt.Errorf("%w: provider_id is required", ErrInvalidRequest)
	}
	if req.ExternalRef == "" && req.ProviderReference == "" {
		return fmt.Errorf("%w: external_ref or provider_reference is required", ErrInvalidRequest)
	}
	return nil
}

func toProviderInitiateRequest(req InitiatePaymentRequest) provider.InitiatePaymentRequest {
	return provider.InitiatePaymentRequest{
		UserID:          req.UserID,
		ExternalRef:     req.ExternalRef,
		AmountMinor:     req.AmountMinor,
		Currency:        req.Currency,
		Country:         req.Country,
		Method:          req.Method,
		Description:     req.Description,
		Customer:        provider.Customer(req.Customer),
		CallbackURL:     req.CallbackURL,
		ReturnURL:       req.ReturnURL,
		ExpiresAt:       req.ExpiresAt,
		Metadata:        cloneStringMap(req.Metadata),
		ProviderOptions: cloneAnyMap(req.ProviderOptions),
	}
}

func toInitiateResult(p provider.Provider, routeResult *routing.RouteResult, resp *provider.InitiatePaymentResponse) *InitiatePaymentResult {
	return &InitiatePaymentResult{
		ProviderID:        resp.ProviderID,
		ProviderName:      p.Name(),
		ExternalRef:       resp.ExternalRef,
		ProviderReference: resp.ProviderReference,
		Status:            resp.Status,
		NextActionType:    resp.NextActionType,
		NextActionURL:     resp.NextActionURL,
		CustomerMessage:   resp.CustomerMessage,
		Route:             routeInfoFrom(routeResult),
		ProviderResponse:  resp.ProviderResponse,
		Metadata:          cloneStringMap(resp.Metadata),
	}
}

func toVerifyResult(resp *provider.VerifyPaymentResponse) *VerifyPaymentResult {
	return &VerifyPaymentResult{
		ProviderID:        resp.ProviderID,
		ExternalRef:       resp.ExternalRef,
		ProviderReference: resp.ProviderReference,
		Status:            resp.Status,
		AmountMinor:       resp.AmountMinor,
		Currency:          resp.Currency,
		PaidAt:            resp.PaidAt,
		ProviderResponse:  resp.ProviderResponse,
		Metadata:          cloneStringMap(resp.Metadata),
	}
}

func routeInfoFrom(result *routing.RouteResult) RouteInfo {
	if result == nil {
		return RouteInfo{}
	}
	info := RouteInfo{
		ProviderID:      result.ProviderID,
		RouteKey:        result.RouteKey,
		SelectionReason: result.SelectionReason,
		CandidateIDs:    append([]provider.ID(nil), result.CandidateIDs...),
		Skipped:         make([]RouteSkip, 0, len(result.Skipped)),
	}
	for _, skip := range result.Skipped {
		info.Skipped = append(info.Skipped, RouteSkip{ProviderID: skip.ProviderID, Reason: skip.Reason})
	}
	return info
}

func enrichProviderOptions(providerID provider.ID, country string, operator MobileMoneyOperator, options map[string]any) map[string]any {
	out := cloneAnyMap(options)
	if out == nil {
		out = map[string]any{}
	}

	operator = normalizeOperator(operator)
	if operator == "" {
		return out
	}

	if _, exists := out["operator"]; !exists {
		out["operator"] = string(operator)
	}
	if _, exists := out["network"]; !exists {
		out["network"] = string(operator)
	}

	switch normalizeProviderID(providerID) {
	case provider.ProviderMoolre:
		if _, exists := out["channel"]; !exists {
			if channel := moolreChannel(operator); channel != "" {
				out["channel"] = channel
			}
		}
	case provider.ProviderPawaPay:
		if _, exists := out["provider"]; !exists {
			if code := pawapayProviderCode(country, operator); code != "" {
				out["provider"] = code
			}
		}
	}

	return out
}

func moolreChannel(operator MobileMoneyOperator) string {
	switch normalizeOperator(operator) {
	case MobileMoneyOperatorMTN:
		return "13"
	case MobileMoneyOperatorTelecel:
		return "6"
	case MobileMoneyOperatorAT:
		return "7"
	default:
		return ""
	}
}

func pawapayProviderCode(country string, operator MobileMoneyOperator) string {
	country = normalizeCountry(country)
	if country != "GH" && country != "GHA" {
		return ""
	}

	switch normalizeOperator(operator) {
	case MobileMoneyOperatorMTN:
		return "MTN_MOMO_GHA"
	case MobileMoneyOperatorTelecel:
		return "VODAFONE_GHA"
	case MobileMoneyOperatorAT:
		return "AIRTELTIGO_GHA"
	default:
		return ""
	}
}

func hasProviderOperatorOptions(options map[string]any) bool {
	for _, key := range []string{"operator", "network", "provider", "channel", "moolre_channel"} {
		if strings.TrimSpace(fmt.Sprint(options[key])) != "" && fmt.Sprint(options[key]) != "<nil>" {
			return true
		}
	}
	return false
}

func normalizeCustomer(customer Customer, fallbackCountry string, normalizePhone bool) Customer {
	customer.Name = strings.TrimSpace(customer.Name)
	customer.Email = strings.TrimSpace(customer.Email)
	customer.Phone = strings.TrimSpace(customer.Phone)
	customer.Country = normalizeCountry(customer.Country)
	if customer.Country == "" {
		customer.Country = fallbackCountry
	}
	if normalizePhone {
		customer.Phone = NormalizePhone(customer.Country, customer.Phone)
	}
	return customer
}

func NormalizePhone(country string, phone string) string {
	country = normalizeCountry(country)
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.TrimPrefix(phone, "+")

	if country == "GH" || country == "GHA" {
		if strings.HasPrefix(phone, "0") && len(phone) == 10 {
			return "233" + phone[1:]
		}
	}

	return phone
}

func normalizeHooks(hooks Hooks) Hooks {
	if hooks == nil {
		return NoopHooks{}
	}
	return hooks
}

func normalizeProviderID(id provider.ID) provider.ID {
	return provider.ID(strings.ToLower(strings.TrimSpace(string(id))))
}

func normalizeCountry(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func normalizePaymentMethod(method provider.PaymentMethod) provider.PaymentMethod {
	return provider.PaymentMethod(strings.ToLower(strings.TrimSpace(string(method))))
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
