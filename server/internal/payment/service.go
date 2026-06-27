package payment

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingTransactionID = errors.New("missing transaction id")
	ErrMissingCountry       = errors.New("missing country")
	ErrMissingNetwork       = errors.New("missing network")
	ErrMissingPhoneNumber   = errors.New("missing phone number")
	ErrMissingAmount        = errors.New("missing amount")
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrMissingCurrency      = errors.New("missing currency")
	ErrMissingRouter        = errors.New("missing payment router")
	ErrNoProviderResolved   = errors.New("no payment provider resolved")
)

type Service struct {
	router Router
}

func NewService(router Router) *Service {
	return &Service{
		router: router,
	}
}

func (s *Service) Charge(ctx context.Context, payload UnifiedPayload) (*ChargeResult, error) {
	payload = normalizePayload(payload)

	if err := validateChargePayload(payload); err != nil {
		return nil, err
	}

	if s.router == nil {
		return nil, ErrMissingRouter
	}

	route, err := s.router.Resolve(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("resolve payment route: %w", err)
	}

	if route == nil || route.Provider == nil {
		return nil, ErrNoProviderResolved
	}

	result, err := route.Provider.Charge(ctx, route.Payload)
	if err != nil {
		return nil, fmt.Errorf("%s charge failed: %w", route.Provider.Name(), err)
	}

	if result == nil {
		return &ChargeResult{
			TransactionID: route.Payload.TransactionID,
			Provider:      route.Provider.Name(),
			Status:        PaymentStatusPending,
			Message:       "payment request accepted",
			CreatedAt:     time.Now().UTC(),
		}, nil
	}

	if result.TransactionID == "" {
		result.TransactionID = route.Payload.TransactionID
	}

	if result.Provider == "" {
		result.Provider = route.Provider.Name()
	}

	if result.Status == "" {
		result.Status = PaymentStatusPending
	}

	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}

	return result, nil
}

func validateChargePayload(payload UnifiedPayload) error {
	if strings.TrimSpace(payload.TransactionID) == "" {
		return ErrMissingTransactionID
	}

	if strings.TrimSpace(payload.Country) == "" {
		return ErrMissingCountry
	}

	if strings.TrimSpace(payload.Network) == "" {
		return ErrMissingNetwork
	}

	if strings.TrimSpace(payload.PhoneNumber) == "" {
		return ErrMissingPhoneNumber
	}

	if strings.TrimSpace(payload.Amount) == "" {
		return ErrMissingAmount
	}

	amount, err := strconv.ParseFloat(payload.Amount, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAmount, payload.Amount)
	}

	if amount <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	if strings.TrimSpace(payload.Currency) == "" {
		return ErrMissingCurrency
	}

	if len(strings.TrimSpace(payload.Currency)) != 3 {
		return fmt.Errorf("invalid currency: %s", payload.Currency)
	}

	return nil
}

func normalizePayload(payload UnifiedPayload) UnifiedPayload {
	payload.TransactionID = strings.TrimSpace(payload.TransactionID)
	payload.Country = strings.ToUpper(strings.TrimSpace(payload.Country))
	payload.Network = strings.ToUpper(strings.TrimSpace(payload.Network))
	payload.PhoneNumber = normalizePhone(payload.PhoneNumber)
	payload.Amount = strings.TrimSpace(payload.Amount)
	payload.Currency = strings.ToUpper(strings.TrimSpace(payload.Currency))
	payload.Operator = strings.TrimSpace(payload.Operator)

	return payload
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.TrimPrefix(phone, "+")

	return phone
}
