package payment

import (
	"context"
	"errors"
	"strings"
	"time"

	paymentkernel "github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/google/uuid"
)

var ErrInvalidPayment = errors.New("invalid payment")

type Processor interface {
	InitiatePayment(ctx context.Context, req paymentkernel.InitiatePaymentRequest) (*paymentkernel.InitiatePaymentResult, error)
}

type TransactionCreator interface {
	Create(ctx context.Context, params transaction.CreateParams) (*transaction.Transaction, error)
}

type CheckoutCompleter interface {
	CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error
}

type Service struct {
	repository        *Repository
	processor         Processor
	transactions      TransactionCreator
	checkoutCompleter CheckoutCompleter
}

func NewService(repository *Repository, processor Processor, transactions TransactionCreator, checkoutCompleter CheckoutCompleter) *Service {
	return &Service{repository: repository, processor: processor, transactions: transactions, checkoutCompleter: checkoutCompleter}
}

func (s *Service) SetProcessor(processor Processor) {
	s.processor = processor
}

func (s *Service) StartCheckoutPayment(ctx context.Context, params StartCheckoutPaymentParams) (*StartCheckoutPaymentResult, error) {
	if s.processor == nil {
		return nil, errors.New("payment processor is not configured")
	}
	if params.UserID == uuid.Nil || params.CheckoutID == uuid.Nil || params.Amount <= 0 || strings.TrimSpace(params.Currency) == "" {
		return nil, ErrInvalidPayment
	}

	externalRef := uuid.NewString()
	metadata := map[string]string{"checkout_session_id": params.CheckoutID.String(), "user_id": params.UserID.String()}
	for key, value := range params.Metadata {
		if _, exists := metadata[key]; !exists {
			metadata[key] = value
		}
	}

	result, err := s.processor.InitiatePayment(ctx, paymentkernel.InitiatePaymentRequest{
		UserID:            params.UserID.String(),
		ExternalRef:       externalRef,
		AmountMinor:       params.Amount,
		Currency:          params.Currency,
		Country:           params.Country,
		Method:            provider.PaymentMethodMobileMoney,
		Operator:          paymentkernel.MobileMoneyOperator(params.Operator),
		PreferredProvider: provider.ID(params.PreferredProvider),
		Description:       params.Label,
		Customer: paymentkernel.Customer{Phone: params.Phone, Country: params.Country, Name: params.CustomerName, Email: params.CustomerEmail},
		ReturnURL:         params.ReturnURL,
		Metadata:          metadata,
	})
	if err != nil {
		return nil, err
	}

	status := statusFromProvider(result.Status)
	checkoutID := params.CheckoutID
	paymentRecord, err := s.repository.Create(ctx, CreateParams{
		UserID:     params.UserID,
		CheckoutID: &checkoutID,
		CustomerID: params.CustomerID,
		ExternalID: result.ExternalRef,
		Provider:   string(result.ProviderID),
		Status:     status,
		Currency:   params.Currency,
		Amount:     params.Amount,
		Metadata:   stringMapToAny(metadata),
	})
	if err != nil {
		return nil, err
	}

	if result.ProviderReference != "" || status != StatusPending {
		paymentRecord, _ = s.repository.UpdateFromProvider(ctx, UpdateFromProviderParams{ExternalID: result.ExternalRef, Provider: string(result.ProviderID), ProviderReference: result.ProviderReference, Status: status, Metadata: stringMapToAny(result.Metadata)})
	}

	return &StartCheckoutPaymentResult{PaymentID: paymentRecord.ID, CheckoutSessionID: params.CheckoutID, ExternalRef: result.ExternalRef, ProviderID: string(result.ProviderID), ProviderReference: result.ProviderReference, Status: status, AttemptStatus: AttemptStatus(result.Status), NextActionType: string(result.NextActionType), NextActionURL: result.NextActionURL, CustomerMessage: result.CustomerMessage}, nil
}

func (s *Service) List(ctx context.Context, params ListParams) ([]Payment, error) {
	return s.repository.List(ctx, params)
}

func (s *Service) PaymentInitiated(ctx paymentkernel.Context, result *paymentkernel.InitiatePaymentResult) error { return nil }
func (s *Service) PaymentVerified(ctx paymentkernel.Context, result *paymentkernel.VerifyPaymentResult) error { return nil }

func (s *Service) WebhookProcessed(ctx paymentkernel.Context, result *paymentkernel.ProcessedWebhookResult) error {
	if result == nil {
		return nil
	}
	metadata := stringMapToAny(result.Metadata)
	if result.Verification != nil && len(result.Verification.Metadata) > 0 {
		metadata = stringMapToAny(result.Verification.Metadata)
	}

	paymentRecord, err := s.repository.UpdateFromProvider(contextFromPayment(ctx), UpdateFromProviderParams{ExternalID: result.ExternalRef, Provider: string(result.ProviderID), ProviderReference: result.ProviderReference, Status: statusFromProvider(result.Status), Metadata: metadata})
	if err != nil {
		return err
	}
	if paymentRecord.Status != StatusCaptured {
		return nil
	}
	if s.transactions != nil {
		externalID := result.ProviderReference
		if strings.TrimSpace(externalID) == "" {
			externalID = result.ExternalRef
		}
		_, _ = s.transactions.Create(contextFromPayment(ctx), transaction.CreateParams{UserID: paymentRecord.UserID, PaymentID: &paymentRecord.ID, CheckoutID: paymentRecord.CheckoutID, ExternalID: externalID, Type: transaction.TypeCapture, Status: transaction.StatusSucceeded, Currency: paymentRecord.Currency, Amount: paymentRecord.Amount, OccurredAt: time.Now().UTC(), Metadata: paymentRecord.Metadata})
	}
	if s.checkoutCompleter != nil && paymentRecord.CheckoutID != nil {
		return s.checkoutCompleter.CompletePaidCheckout(contextFromPayment(ctx), *paymentRecord.CheckoutID)
	}
	return nil
}

func statusFromProvider(status provider.PaymentStatus) Status {
	switch status {
	case provider.PaymentStatusSucceeded:
		return StatusCaptured
	case provider.PaymentStatusFailed:
		return StatusFailed
	case provider.PaymentStatusCanceled, provider.PaymentStatusExpired:
		return StatusVoided
	default:
		return StatusPending
	}
}

func contextFromPayment(ctx paymentkernel.Context) context.Context {
	if realCtx, ok := ctx.(context.Context); ok {
		return realCtx
	}
	return context.Background()
}

var _ paymentkernel.Hooks = (*Service)(nil)
