package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type testSubscriptionRenewer struct {
	called         int
	userID         uuid.UUID
	subscriptionID uuid.UUID
	err            error
}

func (r *testSubscriptionRenewer) RenewPeriodTx(_ context.Context, _ pgx.Tx, userID, subscriptionID uuid.UUID) error {
	r.called++
	r.userID = userID
	r.subscriptionID = subscriptionID
	return r.err
}

type testDunningSettlement struct {
	markPaidCalled int
	revokeCalled   int
	userID         uuid.UUID
	attemptID      uuid.UUID
	subscriptionID uuid.UUID
	checkoutID     uuid.UUID
	tokenID        uuid.UUID
}

func (d *testDunningSettlement) MarkAttemptPaidTx(_ context.Context, _ pgx.Tx, userID, attemptID, subscriptionID, checkoutID uuid.UUID) error {
	d.markPaidCalled++
	d.userID = userID
	d.attemptID = attemptID
	d.subscriptionID = subscriptionID
	d.checkoutID = checkoutID
	return nil
}

func (d *testDunningSettlement) RevokeTokenByIDTx(_ context.Context, _ pgx.Tx, userID, tokenID, attemptID uuid.UUID) error {
	d.revokeCalled++
	d.userID = userID
	d.tokenID = tokenID
	d.attemptID = attemptID
	return nil
}

func TestCompleteDunningRenewalCoordinatesModules(t *testing.T) {
	userID := uuid.New()
	subscriptionID := uuid.New()
	checkoutID := uuid.New()
	attemptID := uuid.New()
	tokenID := uuid.New()
	subscriptions := &testSubscriptionRenewer{}
	dunning := &testDunningSettlement{}
	service := NewService(nil, nil)
	service.SetCompletionServices(subscriptions, dunning, nil)

	err := service.completeDunningRenewal(context.Background(), nil, &checkout.Session{
		ID:             checkoutID,
		UserID:         userID,
		SubscriptionID: &subscriptionID,
		Mode:           checkout.ModeRenewal,
		Source:         checkout.SourceDunning,
		Metadata: map[string]any{
			"dunning_attempt_id": attemptID.String(),
			"dunning_token_id":   tokenID.String(),
		},
	})
	if err != nil {
		t.Fatalf("complete renewal: %v", err)
	}
	if subscriptions.called != 1 {
		t.Fatalf("expected subscription renewal once, got %d", subscriptions.called)
	}
	if dunning.markPaidCalled != 1 {
		t.Fatalf("expected mark paid once, got %d", dunning.markPaidCalled)
	}
	if dunning.revokeCalled != 1 {
		t.Fatalf("expected revoke once, got %d", dunning.revokeCalled)
	}
	if subscriptions.userID != userID || subscriptions.subscriptionID != subscriptionID {
		t.Fatalf("unexpected subscription ids")
	}
	if dunning.attemptID != attemptID || dunning.tokenID != tokenID || dunning.checkoutID != checkoutID {
		t.Fatalf("unexpected dunning ids")
	}
}

func TestCompleteDunningRenewalRequiresMetadataBeforeCallingModules(t *testing.T) {
	subscriptions := &testSubscriptionRenewer{}
	dunning := &testDunningSettlement{}
	service := NewService(nil, nil)
	service.SetCompletionServices(subscriptions, dunning, nil)
	subscriptionID := uuid.New()

	err := service.completeDunningRenewal(context.Background(), nil, &checkout.Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		SubscriptionID: &subscriptionID,
		Mode:           checkout.ModeRenewal,
		Source:         checkout.SourceDunning,
		Metadata: map[string]any{
			"dunning_attempt_id": uuid.NewString(),
		},
	})
	if !errors.Is(err, ErrInvalidCheckoutCompletion) {
		t.Fatalf("expected invalid checkout completion, got %v", err)
	}
	if subscriptions.called != 0 || dunning.markPaidCalled != 0 || dunning.revokeCalled != 0 {
		t.Fatalf("expected no module calls after invalid metadata")
	}
}

func TestCompleteDunningRenewalStopsAfterSubscriptionError(t *testing.T) {
	expectedErr := errors.New("renewal failed")
	subscriptions := &testSubscriptionRenewer{err: expectedErr}
	dunning := &testDunningSettlement{}
	service := NewService(nil, nil)
	service.SetCompletionServices(subscriptions, dunning, nil)
	subscriptionID := uuid.New()

	err := service.completeDunningRenewal(context.Background(), nil, &checkout.Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		SubscriptionID: &subscriptionID,
		Mode:           checkout.ModeRenewal,
		Source:         checkout.SourceDunning,
		Metadata: map[string]any{
			"dunning_attempt_id": uuid.NewString(),
			"dunning_token_id":   uuid.NewString(),
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected renewal error, got %v", err)
	}
	if dunning.markPaidCalled != 0 || dunning.revokeCalled != 0 {
		t.Fatalf("expected no dunning calls after renewal failure")
	}
}
