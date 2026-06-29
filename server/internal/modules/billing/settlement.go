package billing

import (
	"context"
	"strings"
	"time"

	modulepayment "github.com/cuffeyvidzro/leamout/internal/modules/payment"
	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/cuffeyvidzro/leamout/internal/modules/wallet"
	"github.com/google/uuid"
)

type TransactionCreator interface {
	Create(ctx context.Context, params transaction.CreateParams) (*transaction.Transaction, error)
}

type WalletCreditor interface {
	CreditPaymentCapture(ctx context.Context, params wallet.CreditPaymentCaptureParams) error
}

func (s *Service) SetSettlementServices(transactions TransactionCreator, wallet WalletCreditor) {
	s.transactions = transactions
	s.wallet = wallet
}

// SettleCapturedPayment coordinates the business effects of a captured payment.
// Payment remains responsible for recording payment/provider state; billing owns
// the post-capture business flow.
func (s *Service) SettleCapturedPayment(ctx context.Context, paymentRecord *modulepayment.Payment) error {
	if paymentRecord == nil || paymentRecord.Status != modulepayment.StatusCaptured {
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

	if paymentRecord.CheckoutID != nil && *paymentRecord.CheckoutID != uuid.Nil {
		return s.CompletePaidCheckout(ctx, *paymentRecord.CheckoutID)
	}

	return nil
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}
