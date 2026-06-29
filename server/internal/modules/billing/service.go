package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCheckoutCompletion = errors.New("invalid checkout completion")
	ErrCheckoutNotFound         = errors.New("checkout session not found")
)

type CheckoutStateRepository interface {
	LockByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*checkout.Session, error)
	CompleteTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*checkout.Session, error)
}

type SubscriptionRenewer interface {
	RenewPeriodTx(ctx context.Context, tx pgx.Tx, userID, subscriptionID uuid.UUID) error
}

type DunningSettlementRepository interface {
	MarkAttemptPaidTx(ctx context.Context, tx pgx.Tx, userID, attemptID, subscriptionID, checkoutID uuid.UUID) error
	RevokeTokenByIDTx(ctx context.Context, tx pgx.Tx, userID, tokenID, attemptID uuid.UUID) error
}

type BenefitGranter interface {
	GrantSubscriptionBenefitsTx(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error
}

type UsageCreditApplier interface {
	ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error
}

type Service struct {
	db                 *pgxpool.Pool
	checkoutState      CheckoutStateRepository
	subscriptionRenewal SubscriptionRenewer
	dunningSettlement  DunningSettlementRepository
	benefitGrants      BenefitGranter
	usageCredits       UsageCreditApplier
	transactions       TransactionCreator
	wallet             WalletCreditor
}

func NewService(db *pgxpool.Pool, checkoutState CheckoutStateRepository, usageCredits ...UsageCreditApplier) *Service {
	service := &Service{db: db, checkoutState: checkoutState}
	if len(usageCredits) > 0 {
		service.usageCredits = usageCredits[0]
	}
	return service
}

func (s *Service) SetCompletionServices(subscriptionRenewal SubscriptionRenewer, dunningSettlement DunningSettlementRepository, benefitGrants BenefitGranter) {
	s.subscriptionRenewal = subscriptionRenewal
	s.dunningSettlement = dunningSettlement
	s.benefitGrants = benefitGrants
}

// CompletePaidCheckout owns the business flow that happens after a checkout has
// been paid. Checkout remains responsible for checkout state; billing coordinates
// cross-module effects such as subscription renewal, dunning settlement, benefit
// grants, and usage-credit grants.
func (s *Service) CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error {
	if s.db == nil {
		return errors.New("billing database is not configured")
	}
	if s.checkoutState == nil {
		return errors.New("checkout state repository is not configured")
	}
	if checkoutID == uuid.Nil {
		return fmt.Errorf("%w: checkout id is required", ErrInvalidCheckoutCompletion)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin billing checkout completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := s.checkoutState.LockByIDTx(ctx, tx, checkoutID)
	if err != nil {
		return checkoutStateError("lock checkout session for billing completion", err)
	}
	if session.Status == checkout.StatusCompleted {
		if err := s.fulfillSubscriptionBenefits(ctx, tx, session); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if session.Status != checkout.StatusOpen {
		return tx.Commit(ctx)
	}

	session, err = s.checkoutState.CompleteTx(ctx, tx, checkoutID)
	if err != nil {
		return checkoutStateError("complete checkout state", err)
	}
	if isDunningRenewal(session) {
		if err := s.completeDunningRenewal(ctx, tx, session); err != nil {
			return err
		}
	}
	if err := s.fulfillSubscriptionBenefits(ctx, tx, session); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit billing checkout completion: %w", err)
	}
	return nil
}

func (s *Service) completeDunningRenewal(ctx context.Context, tx pgx.Tx, session *checkout.Session) error {
	if !isDunningRenewal(session) {
		return nil
	}
	if s.subscriptionRenewal == nil {
		return errors.New("subscription renewal repository is not configured")
	}
	if s.dunningSettlement == nil {
		return errors.New("dunning settlement repository is not configured")
	}

	attemptID, err := metadataUUID(session.Metadata, "dunning_attempt_id")
	if err != nil {
		return err
	}
	tokenID, err := metadataUUID(session.Metadata, "dunning_token_id")
	if err != nil {
		return err
	}

	if err := s.subscriptionRenewal.RenewPeriodTx(ctx, tx, session.UserID, *session.SubscriptionID); err != nil {
		return fmt.Errorf("renew subscription period from billing checkout completion: %w", err)
	}
	if err := s.dunningSettlement.MarkAttemptPaidTx(ctx, tx, session.UserID, attemptID, *session.SubscriptionID, session.ID); err != nil {
		return fmt.Errorf("mark dunning attempt paid from billing checkout completion: %w", err)
	}
	if err := s.dunningSettlement.RevokeTokenByIDTx(ctx, tx, session.UserID, tokenID, attemptID); err != nil {
		return fmt.Errorf("revoke dunning token from billing checkout completion: %w", err)
	}

	return nil
}

func (s *Service) fulfillSubscriptionBenefits(ctx context.Context, tx pgx.Tx, session *checkout.Session) error {
	if session == nil || session.SubscriptionID == nil {
		return nil
	}
	if s.benefitGrants == nil {
		return errors.New("benefit grant repository is not configured")
	}

	if err := s.benefitGrants.GrantSubscriptionBenefitsTx(ctx, tx, session.UserID, *session.SubscriptionID, session.ID, session.CustomerID); err != nil {
		return fmt.Errorf("fulfill subscription benefits from billing checkout completion: %w", err)
	}

	if s.usageCredits != nil {
		if err := s.usageCredits.ApplySubscriptionCredits(ctx, tx, session.UserID, *session.SubscriptionID, session.ID, session.CustomerID); err != nil {
			return err
		}
	}

	return nil
}

func checkoutStateError(action string, err error) error {
	if errors.Is(err, checkout.ErrNotFound) {
		return ErrCheckoutNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func isDunningRenewal(session *checkout.Session) bool {
	return session != nil && session.Mode == checkout.ModeRenewal && session.Source == checkout.SourceDunning && session.SubscriptionID != nil
}

func metadataUUID(metadata map[string]any, key string) (uuid.UUID, error) {
	raw, ok := metadata[key]
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: missing %s", ErrInvalidCheckoutCompletion, key)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrInvalidCheckoutCompletion, key)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrInvalidCheckoutCompletion, key)
	}
	return id, nil
}
