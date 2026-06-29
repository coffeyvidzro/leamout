package payment

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/cuffeyvidzro/leamout/internal/modules/wallet"
	corepayment "github.com/cuffeyvidzro/leamout/internal/payment"
	paymentsm "github.com/cuffeyvidzro/leamout/internal/platform/statemachine/payment"
	"github.com/google/uuid"
)

var ErrInvalidPayment = errors.New("invalid payment")

const (
	metadataCheckoutUserID     = "checkout_user_id"
	metadataCheckoutSessionID  = "checkout_session_id"
	metadataCheckoutCustomerID = "checkout_customer_id"
	metadataCheckoutFeeAmount  = "checkout_fee_amount"
)

type Charger interface {
	Charge(ctx context.Context, payload corepayment.UnifiedPayload) (*corepayment.ChargeResult, error)
}

type TransactionCreator interface {
	Create(ctx context.Context, params transaction.CreateParams) (*transaction.Transaction, error)
}

type WalletCreditor interface {
	CreditPaymentCapture(ctx context.Context, params wallet.CreditPaymentCaptureParams) error
}

type CheckoutCompleter interface {
	CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error
}

type Service struct {
	repository        *Repository
	charger           Charger
	transactions      TransactionCreator
	wallet            WalletCreditor
	checkoutCompleter CheckoutCompleter
}

func NewService(repository *Repository, charger Charger, transactions TransactionCreator, wallet WalletCreditor) *Service {
	return &Service{
		repository:   repository,
		charger:      charger,
		transactions: transactions,
		wallet:       wallet,
	}
}

func (s *Service) SetCharger(charger Charger) {
	s.charger = charger
}

func (s *Service) SetCheckoutCompleter(completer CheckoutCompleter) {
	s.checkoutCompleter = completer
}

// Charge starts a checkout payment through the internal payment orchestration layer
// and records the business payment + provider attempt.
func (s *Service) Charge(ctx context.Context, payload corepayment.UnifiedPayload) (*corepayment.ChargeResult, error) {
	if s.repository == nil {
		return nil, errors.New("payment repository is not configured")
	}
	if s.charger == nil {
		return nil, errors.New("payment charger is not configured")
	}

	payload = normalizeCorePayload(payload)

	paymentCtx, err := checkoutContextFromMetadata(payload.Metadata)
	if err != nil {
		return nil, err
	}

	amount, err := parseAmount(payload.Amount)
	if err != nil {
		return nil, err
	}

	feeAmount, err := feeAmountFromMetadata(payload.Metadata)
	if err != nil {
		return nil, err
	}
	if feeAmount > amount {
		return nil, fmt.Errorf("%w: fee amount cannot be greater than amount", ErrInvalidPayment)
	}

	result, chargeErr := s.charger.Charge(ctx, payload)
	if chargeErr != nil {
		return nil, chargeErr
	}
	if result == nil {
		result = &corepayment.ChargeResult{
			TransactionID: payload.TransactionID,
			Status:        corepayment.PaymentStatusPending,
			Message:       "payment request accepted",
			CreatedAt:     time.Now().UTC(),
		}
	}

	if result.TransactionID == "" {
		result.TransactionID = payload.TransactionID
	}

	provider := strings.ToLower(strings.TrimSpace(string(result.Provider)))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(string(payload.Provider)))
	}

	metadata := paymentMetadata(payload, result, paymentCtx, feeAmount)

	paymentRecord, err := s.repository.Create(ctx, CreateParams{
		UserID:            paymentCtx.UserID,
		CheckoutID:        &paymentCtx.CheckoutID,
		CustomerID:        paymentCtx.CustomerID,
		ExternalID:        result.TransactionID,
		Provider:          provider,
		ProviderReference: result.ProviderReference,
		Status:            statusFromCore(result.Status),
		Currency:          payload.Currency,
		Amount:            amount,
		FeeAmount:         feeAmount,
		Metadata:          metadata,
	})
	if err != nil {
		return nil, err
	}

	_, _ = s.repository.CreateAttempt(ctx, CreateAttemptParams{
		PaymentID:         paymentRecord.ID,
		Provider:          provider,
		ProviderReference: result.ProviderReference,
		Status:            attemptStatusFromCore(result.Status),
		RawRequest:        attemptRequest(payload),
		RawResponse:       attemptResponse(result),
	})

	if paymentRecord.Status == StatusCaptured {
		if err := s.settleCapturedPayment(ctx, paymentRecord); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Payment, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrInvalidPayment
	}

	return s.repository.Get(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, params ListParams) ([]Payment, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidPayment
	}

	return s.repository.List(ctx, params)
}

func (s *Service) ApplyProviderResult(ctx context.Context, params UpdateFromProviderParams) (*Payment, error) {
	if s.repository == nil {
		return nil, errors.New("payment repository is not configured")
	}

	params.ExternalID = strings.TrimSpace(params.ExternalID)
	params.Provider = strings.ToLower(strings.TrimSpace(params.Provider))
	if params.ExternalID == "" || params.Provider == "" || params.Status == "" {
		return nil, ErrInvalidPayment
	}

	currentPayment, err := s.repository.GetByProviderExternalID(ctx, params.Provider, params.ExternalID)
	if err != nil {
		return nil, err
	}
	if !paymentsm.CanTransition(paymentsm.Status(currentPayment.Status), paymentsm.Status(params.Status)) {
		return nil, fmt.Errorf("%w: cannot transition payment from %s to %s", ErrInvalidPayment, currentPayment.Status, params.Status)
	}

	paymentRecord, err := s.repository.UpdateFromProvider(ctx, params)
	if err != nil {
		return nil, err
	}

	_, _ = s.repository.CreateAttempt(ctx, CreateAttemptParams{
		PaymentID:         paymentRecord.ID,
		Provider:          params.Provider,
		ProviderReference: params.ProviderReference,
		Status:            attemptStatusFromPayment(params.Status),
		RawResponse:       params.Metadata,
	})

	if paymentRecord.Status == StatusCaptured {
		if err := s.settleCapturedPayment(ctx, paymentRecord); err != nil {
			return nil, err
		}
	}

	return paymentRecord, nil
}

func (s *Service) settleCapturedPayment(ctx context.Context, paymentRecord *Payment) error {
	if paymentRecord == nil || paymentRecord.Status != StatusCaptured {
		return nil
	}

	var transactionRecord *transaction.Transaction
	var err error

	if s.transactions != nil {
		externalID := paymentRecord.ExternalID
		if paymentRecord.ProviderReference != nil && strings.TrimSpace(*paymentRecord.ProviderReference) != "" {
			externalID = strings.TrimSpace(*paymentRecord.ProviderReference)
		}

		transactionRecord, err = s.transactions.Create(ctx, transaction.CreateParams{
			UserID:     paymentRecord.UserID,
			PaymentID:  &paymentRecord.ID,
			CheckoutID: paymentRecord.CheckoutID,
			ExternalID: externalID,
			Type:       transaction.TypeCapture,
			Status:     transaction.StatusSucceeded,
			Currency:   paymentRecord.Currency,
			Amount:     paymentRecord.NetAmount,
			OccurredAt: time.Now().UTC(),
			Metadata:   paymentRecord.Metadata,
		})
		if err != nil {
			return err
		}
	}

	if s.wallet != nil && transactionRecord != nil {
		if err := s.wallet.CreditPaymentCapture(ctx, wallet.CreditPaymentCaptureParams{
			UserID:        paymentRecord.UserID,
			PaymentID:     paymentRecord.ID,
			TransactionID: transactionRecord.ID,
			Country:       metadataString(paymentRecord.Metadata, "payment_country"),
			Currency:      paymentRecord.Currency,
			Amount:        paymentRecord.NetAmount,
			Metadata:      paymentRecord.Metadata,
		}); err != nil {
			return err
		}
	}

	if s.checkoutCompleter != nil && paymentRecord.CheckoutID != nil {
		return s.checkoutCompleter.CompletePaidCheckout(ctx, *paymentRecord.CheckoutID)
	}

	return nil
}

type checkoutPaymentContext struct {
	UserID     uuid.UUID
	CheckoutID uuid.UUID
	CustomerID *uuid.UUID
}

func checkoutContextFromMetadata(metadata map[string]string) (*checkoutPaymentContext, error) {
	if metadata == nil {
		return nil, fmt.Errorf("%w: missing checkout metadata", ErrInvalidPayment)
	}

	userID, err := metadataUUID(metadata, metadataCheckoutUserID)
	if err != nil {
		return nil, err
	}
	checkoutID, err := metadataUUID(metadata, metadataCheckoutSessionID)
	if err != nil {
		return nil, err
	}

	var customerID *uuid.UUID
	if raw := strings.TrimSpace(metadata[metadataCheckoutCustomerID]); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid %s", ErrInvalidPayment, metadataCheckoutCustomerID)
		}
		customerID = &parsed
	}

	return &checkoutPaymentContext{
		UserID:     userID,
		CheckoutID: checkoutID,
		CustomerID: customerID,
	}, nil
}

func metadataUUID(metadata map[string]string, key string) (uuid.UUID, error) {
	raw := strings.TrimSpace(metadata[key])
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%w: missing %s", ErrInvalidPayment, key)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrInvalidPayment, key)
	}

	return id, nil
}

func feeAmountFromMetadata(metadata map[string]string) (int64, error) {
	if metadata == nil {
		return 0, nil
	}

	raw := strings.TrimSpace(metadata[metadataCheckoutFeeAmount])
	if raw == "" {
		return 0, nil
	}

	feeAmount, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || feeAmount < 0 {
		return 0, fmt.Errorf("%w: invalid %s", ErrInvalidPayment, metadataCheckoutFeeAmount)
	}

	return feeAmount, nil
}

func parseAmount(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%w: missing amount", ErrInvalidPayment)
	}

	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid amount", ErrInvalidPayment)
	}
	if amount <= 0 {
		return 0, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidPayment)
	}

	return amount, nil
}

func normalizeCorePayload(payload corepayment.UnifiedPayload) corepayment.UnifiedPayload {
	payload.TransactionID = strings.TrimSpace(payload.TransactionID)
	payload.Country = strings.ToUpper(strings.TrimSpace(payload.Country))
	payload.Network = strings.ToUpper(strings.TrimSpace(payload.Network))
	payload.PhoneNumber = strings.TrimPrefix(strings.TrimSpace(payload.PhoneNumber), "+")
	payload.Amount = strings.TrimSpace(payload.Amount)
	payload.Currency = strings.ToUpper(strings.TrimSpace(payload.Currency))
	payload.Operator = strings.TrimSpace(payload.Operator)

	if payload.Metadata == nil {
		payload.Metadata = map[string]string{}
	}

	return payload
}

func statusFromCore(status corepayment.PaymentStatus) Status {
	switch status {
	case corepayment.PaymentStatusSucceeded:
		return StatusCaptured
	case corepayment.PaymentStatusFailed:
		return StatusFailed
	default:
		return StatusPending
	}
}

func attemptStatusFromCore(status corepayment.PaymentStatus) AttemptStatus {
	switch status {
	case corepayment.PaymentStatusSucceeded:
		return AttemptStatusSucceeded
	case corepayment.PaymentStatusFailed:
		return AttemptStatusFailed
	default:
		return AttemptStatusProcessing
	}
}

func attemptStatusFromPayment(status Status) AttemptStatus {
	switch status {
	case StatusCaptured:
		return AttemptStatusSucceeded
	case StatusFailed:
		return AttemptStatusFailed
	case StatusVoided:
		return AttemptStatusCanceled
	default:
		return AttemptStatusProcessing
	}
}

func attemptRequest(payload corepayment.UnifiedPayload) map[string]any {
	return map[string]any{
		"transaction_id": payload.TransactionID,
		"country":        payload.Country,
		"network":        payload.Network,
		"phone_number":   payload.PhoneNumber,
		"amount":         payload.Amount,
		"currency":       payload.Currency,
		"operator":       payload.Operator,
		"provider":       string(payload.Provider),
	}
}

func attemptResponse(result *corepayment.ChargeResult) map[string]any {
	if result == nil {
		return map[string]any{}
	}

	return map[string]any{
		"transaction_id":     result.TransactionID,
		"provider":           string(result.Provider),
		"provider_reference": result.ProviderReference,
		"status":             string(result.Status),
		"message":            result.Message,
		"created_at":         result.CreatedAt,
	}
}

func paymentMetadata(
	payload corepayment.UnifiedPayload,
	result *corepayment.ChargeResult,
	paymentCtx *checkoutPaymentContext,
	feeAmount int64,
) map[string]any {
	metadata := map[string]any{
		"checkout_session_id":    paymentCtx.CheckoutID.String(),
		"payment_transaction_id": result.TransactionID,
		"payment_provider":       string(result.Provider),
		"payment_status":         string(result.Status),
		"payment_country":        payload.Country,
		"payment_network":        payload.Network,
		"payment_currency":       payload.Currency,
		"payment_phone":          payload.PhoneNumber,
		"payment_amount":         payload.Amount,
	}

	if paymentCtx.CustomerID != nil {
		metadata["checkout_customer_id"] = paymentCtx.CustomerID.String()
	}
	if feeAmount > 0 {
		metadata["payment_fee_amount"] = feeAmount
	}
	if result.ProviderReference != "" {
		metadata["payment_provider_reference"] = result.ProviderReference
	}
	if result.Message != "" {
		metadata["payment_message"] = result.Message
	}

	return metadata
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}
