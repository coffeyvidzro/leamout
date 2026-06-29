package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	modulepayment "github.com/cuffeyvidzro/leamout/internal/modules/payment"
	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/cuffeyvidzro/leamout/internal/modules/wallet"
	"github.com/google/uuid"
)

type recordingTransactionCreator struct {
	params transaction.CreateParams
	called int
	err    error
}

func (r *recordingTransactionCreator) Create(_ context.Context, params transaction.CreateParams) (*transaction.Transaction, error) {
	r.called++
	r.params = params
	if r.err != nil {
		return nil, r.err
	}
	externalID := params.ExternalID
	return &transaction.Transaction{
		ID:         uuid.New(),
		UserID:     params.UserID,
		PaymentID:  params.PaymentID,
		CheckoutID: params.CheckoutID,
		ExternalID: &externalID,
		Type:       params.Type,
		Status:     params.Status,
		Currency:   params.Currency,
		Amount:     params.Amount,
		OccurredAt: params.OccurredAt,
		Metadata:   params.Metadata,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

type recordingWalletCreditor struct {
	params wallet.CreditPaymentCaptureParams
	called int
	err    error
}

func (r *recordingWalletCreditor) CreditPaymentCapture(_ context.Context, params wallet.CreditPaymentCaptureParams) error {
	r.called++
	r.params = params
	return r.err
}

func TestSettleCapturedPaymentCreatesCaptureAndCreditsWallet(t *testing.T) {
	paymentID := uuid.New()
	userID := uuid.New()
	checkoutID := uuid.New()
	providerReference := "provider-ref-123"
	transactions := &recordingTransactionCreator{}
	wallets := &recordingWalletCreditor{}
	service := NewService(nil, nil)
	service.SetSettlementServices(transactions, wallets)

	err := service.SettleCapturedPayment(context.Background(), &modulepayment.Payment{
		ID:                paymentID,
		UserID:            userID,
		CheckoutID:        &checkoutID,
		ExternalID:        "external-id-123",
		ProviderReference: &providerReference,
		Status:            modulepayment.StatusCaptured,
		Currency:          "GHS",
		Amount:            5129,
		FeeAmount:         129,
		NetAmount:         5000,
		Metadata: map[string]any{
			"payment_country": "GHA",
		},
	})
	if err == nil || !errors.Is(err, ErrCheckoutNotFound) && err.Error() != "billing database is not configured" {
		t.Fatalf("expected checkout completion to fail after settlement without db, got %v", err)
	}

	if transactions.called != 1 {
		t.Fatalf("expected one capture transaction, got %d", transactions.called)
	}
	if transactions.params.UserID != userID {
		t.Fatalf("expected transaction user %s, got %s", userID, transactions.params.UserID)
	}
	if transactions.params.PaymentID == nil || *transactions.params.PaymentID != paymentID {
		t.Fatalf("expected transaction payment %s, got %v", paymentID, transactions.params.PaymentID)
	}
	if transactions.params.CheckoutID == nil || *transactions.params.CheckoutID != checkoutID {
		t.Fatalf("expected transaction checkout %s, got %v", checkoutID, transactions.params.CheckoutID)
	}
	if transactions.params.ExternalID != providerReference {
		t.Fatalf("expected provider reference external id %q, got %q", providerReference, transactions.params.ExternalID)
	}
	if transactions.params.Type != transaction.TypeCapture {
		t.Fatalf("expected transaction type %s, got %s", transaction.TypeCapture, transactions.params.Type)
	}
	if transactions.params.Status != transaction.StatusSucceeded {
		t.Fatalf("expected transaction status %s, got %s", transaction.StatusSucceeded, transactions.params.Status)
	}
	if transactions.params.Amount != 5000 {
		t.Fatalf("expected net capture amount 5000, got %d", transactions.params.Amount)
	}

	if wallets.called != 1 {
		t.Fatalf("expected one wallet credit, got %d", wallets.called)
	}
	if wallets.params.UserID != userID {
		t.Fatalf("expected wallet user %s, got %s", userID, wallets.params.UserID)
	}
	if wallets.params.PaymentID != paymentID {
		t.Fatalf("expected wallet payment %s, got %s", paymentID, wallets.params.PaymentID)
	}
	if wallets.params.Amount != 5000 {
		t.Fatalf("expected wallet credit amount 5000, got %d", wallets.params.Amount)
	}
	if wallets.params.Country != "GHA" {
		t.Fatalf("expected wallet country GHA, got %q", wallets.params.Country)
	}
}

func TestSettleCapturedPaymentWithoutCheckoutOnlySettlesMoney(t *testing.T) {
	transactions := &recordingTransactionCreator{}
	wallets := &recordingWalletCreditor{}
	service := NewService(nil, nil)
	service.SetSettlementServices(transactions, wallets)

	if err := service.SettleCapturedPayment(context.Background(), &modulepayment.Payment{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		ExternalID: "external-id-123",
		Status:     modulepayment.StatusCaptured,
		Currency:   "GHS",
		NetAmount:  5000,
	}); err != nil {
		t.Fatalf("settle captured payment without checkout: %v", err)
	}
	if transactions.called != 1 {
		t.Fatalf("expected one capture transaction, got %d", transactions.called)
	}
	if wallets.called != 1 {
		t.Fatalf("expected one wallet credit, got %d", wallets.called)
	}
}

func TestSettleCapturedPaymentSkipsNonCapturedPayment(t *testing.T) {
	transactions := &recordingTransactionCreator{}
	wallets := &recordingWalletCreditor{}
	service := NewService(nil, nil)
	service.SetSettlementServices(transactions, wallets)

	if err := service.SettleCapturedPayment(context.Background(), &modulepayment.Payment{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Status: modulepayment.StatusPending,
	}); err != nil {
		t.Fatalf("settle non-captured payment: %v", err)
	}
	if transactions.called != 0 {
		t.Fatalf("expected no transaction for non-captured payment, got %d", transactions.called)
	}
	if wallets.called != 0 {
		t.Fatalf("expected no wallet credit for non-captured payment, got %d", wallets.called)
	}
}

func TestSettleCapturedPaymentStopsWhenTransactionFails(t *testing.T) {
	expectedErr := errors.New("transaction failed")
	transactions := &recordingTransactionCreator{err: expectedErr}
	wallets := &recordingWalletCreditor{}
	service := NewService(nil, nil)
	service.SetSettlementServices(transactions, wallets)

	err := service.SettleCapturedPayment(context.Background(), &modulepayment.Payment{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		ExternalID: "external-id-123",
		Status:     modulepayment.StatusCaptured,
		Currency:   "GHS",
		NetAmount:  5000,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected transaction error, got %v", err)
	}
	if wallets.called != 0 {
		t.Fatalf("expected no wallet credit after transaction failure, got %d", wallets.called)
	}
}
