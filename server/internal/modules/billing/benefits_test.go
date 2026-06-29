package billing

import (
	"context"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type testBenefitGranter struct {
	called             int
	userID             uuid.UUID
	subscriptionID     uuid.UUID
	checkoutID         uuid.UUID
	fallbackCustomerID *uuid.UUID
}

func (g *testBenefitGranter) GrantSubscriptionBenefitsTx(_ context.Context, _ pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	g.called++
	g.userID = userID
	g.subscriptionID = subscriptionID
	g.checkoutID = checkoutID
	g.fallbackCustomerID = fallbackCustomerID
	return nil
}

type testUsageApplier struct {
	called             int
	userID             uuid.UUID
	subscriptionID     uuid.UUID
	checkoutID         uuid.UUID
	fallbackCustomerID *uuid.UUID
}

func (a *testUsageApplier) ApplySubscriptionCredits(_ context.Context, _ pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	a.called++
	a.userID = userID
	a.subscriptionID = subscriptionID
	a.checkoutID = checkoutID
	a.fallbackCustomerID = fallbackCustomerID
	return nil
}

func TestFulfillSubscriptionBenefitsCoordinatesModules(t *testing.T) {
	userID := uuid.New()
	subscriptionID := uuid.New()
	checkoutID := uuid.New()
	customerID := uuid.New()
	benefits := &testBenefitGranter{}
	usage := &testUsageApplier{}
	service := NewService(nil, nil, usage)
	service.SetCompletionServices(nil, nil, benefits)

	err := service.fulfillSubscriptionBenefits(context.Background(), nil, &checkout.Session{
		ID:             checkoutID,
		UserID:         userID,
		CustomerID:     &customerID,
		SubscriptionID: &subscriptionID,
	})
	if err != nil {
		t.Fatalf("fulfill subscription benefits: %v", err)
	}
	if benefits.called != 1 {
		t.Fatalf("expected benefits once, got %d", benefits.called)
	}
	if usage.called != 1 {
		t.Fatalf("expected usage once, got %d", usage.called)
	}
	if benefits.userID != userID || benefits.subscriptionID != subscriptionID || benefits.checkoutID != checkoutID {
		t.Fatalf("unexpected benefit ids")
	}
	if usage.userID != userID || usage.subscriptionID != subscriptionID || usage.checkoutID != checkoutID {
		t.Fatalf("unexpected usage ids")
	}
	if benefits.fallbackCustomerID == nil || *benefits.fallbackCustomerID != customerID {
		t.Fatalf("expected benefit customer %s, got %v", customerID, benefits.fallbackCustomerID)
	}
	if usage.fallbackCustomerID == nil || *usage.fallbackCustomerID != customerID {
		t.Fatalf("expected usage customer %s, got %v", customerID, usage.fallbackCustomerID)
	}
}

func TestFulfillSubscriptionBenefitsSkipsWithoutSubscription(t *testing.T) {
	benefits := &testBenefitGranter{}
	usage := &testUsageApplier{}
	service := NewService(nil, nil, usage)
	service.SetCompletionServices(nil, nil, benefits)

	if err := service.fulfillSubscriptionBenefits(context.Background(), nil, &checkout.Session{ID: uuid.New()}); err != nil {
		t.Fatalf("fulfill without subscription: %v", err)
	}
	if benefits.called != 0 {
		t.Fatalf("expected no benefit call, got %d", benefits.called)
	}
	if usage.called != 0 {
		t.Fatalf("expected no usage call, got %d", usage.called)
	}
}
