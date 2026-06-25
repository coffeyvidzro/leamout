package sms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/sms"
	"github.com/cuffeyvidzro/leamout/internal/sms/outbox"
	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
	"github.com/cuffeyvidzro/leamout/internal/sms/provider/mock"
	"github.com/cuffeyvidzro/leamout/internal/sms/routing"
	"github.com/google/uuid"
)

type fakeCredits struct {
	debits  []credits.DebitParams
	refunds []credits.RefundParams
	err     error
}

func (c *fakeCredits) Debit(_ context.Context, params credits.DebitParams) (*credits.Balance, error) {
	c.debits = append(c.debits, params)
	if c.err != nil {
		return nil, c.err
	}
	return &credits.Balance{UserID: params.UserID, Balance: 1000 - params.Amount, Currency: credits.CurrencyGHS}, nil
}

func (c *fakeCredits) Refund(_ context.Context, params credits.RefundParams) (*credits.Balance, error) {
	c.refunds = append(c.refunds, params)
	return &credits.Balance{UserID: params.UserID, Balance: 1000 + params.Amount, Currency: credits.CurrencyGHS}, nil
}

type fakeOutbox struct {
	message  *outbox.Message
	created  []outbox.CreateParams
	debited  int
	sent     int
	refunded int
}

func (o *fakeOutbox) CreateOrGet(_ context.Context, params outbox.CreateParams) (*outbox.Message, bool, error) {
	o.created = append(o.created, params)
	if o.message == nil {
		o.message = &outbox.Message{ID: uuid.New(), Status: outbox.StatusPending}
		return o.message, true, nil
	}

	return o.message, false, nil
}

func (o *fakeOutbox) MarkDebited(_ context.Context, _ uuid.UUID) error {
	o.debited++
	if o.message != nil {
		o.message.Status = outbox.StatusDebited
	}
	return nil
}

func (o *fakeOutbox) MarkSent(_ context.Context, _ uuid.UUID) error {
	o.sent++
	if o.message != nil {
		o.message.Status = outbox.StatusSent
	}
	return nil
}

func (o *fakeOutbox) MarkRefunded(_ context.Context, _ uuid.UUID, _ error) error {
	o.refunded++
	if o.message != nil {
		o.message.Status = outbox.StatusRefunded
	}
	return nil
}

func TestServiceRoutesDebitsAppliesHardcodedSenderAndTrimsMessage(t *testing.T) {
	userID := uuid.New()
	creditSvc := &fakeCredits{}
	client := mock.NewClient(false)
	service := sms.NewService(
		creditSvc,
		routing.NewService(),
		map[string]provider.Provider{routing.ProviderMock: mock.NewProvider(client)},
		sms.Config{DefaultFrom: "Leamout"},
	)

	err := service.Send(context.Background(), sms.Message{
		UserID:  userID,
		To:      " +234 801-234-5678 ",
		From:    "Not Leamout",
		Content: " Renew here ",
	})
	if err != nil {
		t.Fatalf("send sms: %v", err)
	}

	messages := client.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected one message, got %d", len(messages))
	}
	if messages[0].Message.From != "Leamout" {
		t.Fatalf("expected hardcoded sender, got %q", messages[0].Message.From)
	}
	if messages[0].Message.To != "+2348012345678" {
		t.Fatalf("expected normalized recipient, got %q", messages[0].Message.To)
	}
	if messages[0].Message.Content != "Renew here" {
		t.Fatalf("expected trimmed content, got %q", messages[0].Message.Content)
	}
	if len(creditSvc.debits) != 1 {
		t.Fatalf("expected one credit debit, got %d", len(creditSvc.debits))
	}
	if creditSvc.debits[0].Amount != 15 || creditSvc.debits[0].Provider != routing.ProviderMock {
		t.Fatalf("unexpected debit route/cost: %+v", creditSvc.debits[0])
	}
}

func TestServiceValidatesRequiredFields(t *testing.T) {
	service := sms.NewService(&fakeCredits{}, routing.NewService(), nil, sms.Config{DefaultFrom: "Leamout"})
	userID := uuid.New()

	if err := service.Send(context.Background(), sms.Message{Content: "hello"}); !errors.Is(err, sms.ErrUserRequired) {
		t.Fatalf("expected user error, got %v", err)
	}
	if err := service.Send(context.Background(), sms.Message{UserID: userID, Content: "hello"}); !errors.Is(err, sms.ErrRecipientRequired) {
		t.Fatalf("expected recipient error, got %v", err)
	}
	if err := service.Send(context.Background(), sms.Message{UserID: userID, To: "+2348012345678"}); !errors.Is(err, sms.ErrContentRequired) {
		t.Fatalf("expected content error, got %v", err)
	}
}

func TestServiceDebitsBeforeProviderSendAndRefundsOnFailure(t *testing.T) {
	userID := uuid.New()
	creditSvc := &fakeCredits{}
	service := sms.NewService(
		creditSvc,
		routing.NewService(),
		map[string]provider.Provider{routing.ProviderMock: mock.NewProvider(mock.NewClient(true))},
		sms.Config{DefaultFrom: "Leamout"},
	)

	err := service.Send(context.Background(), sms.Message{UserID: userID, To: "+2348012345678", Content: "hello"})
	if err == nil {
		t.Fatal("expected provider send error")
	}
	if len(creditSvc.debits) != 1 {
		t.Fatalf("expected debit before send, got %d", len(creditSvc.debits))
	}
	if len(creditSvc.refunds) != 1 {
		t.Fatalf("expected refund after failed send, got %d", len(creditSvc.refunds))
	}
}

func TestServiceDoesNotSendWhenCreditsAreInsufficient(t *testing.T) {
	userID := uuid.New()
	creditSvc := &fakeCredits{err: credits.ErrInsufficientBalance}
	client := mock.NewClient(false)
	service := sms.NewService(
		creditSvc,
		routing.NewService(),
		map[string]provider.Provider{routing.ProviderMock: mock.NewProvider(client)},
		sms.Config{DefaultFrom: "Leamout"},
	)

	err := service.Send(context.Background(), sms.Message{UserID: userID, To: "+2348012345678", Content: "hello"})
	if !errors.Is(err, credits.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance error, got %v", err)
	}
	if len(client.Messages()) != 0 {
		t.Fatal("expected no provider send when credit debit fails")
	}
}

func TestServiceUsesOutboxForIdempotentSentMessages(t *testing.T) {
	userID := uuid.New()
	creditSvc := &fakeCredits{}
	outboxStore := &fakeOutbox{}
	client := mock.NewClient(false)
	service := sms.NewService(
		creditSvc,
		routing.NewService(),
		map[string]provider.Provider{routing.ProviderMock: mock.NewProvider(client)},
		sms.Config{Outbox: outboxStore},
	)

	msg := sms.Message{UserID: userID, To: "+2348012345678", Content: "hello", Reference: "dunning:attempt-1"}
	if err := service.Send(context.Background(), msg); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := service.Send(context.Background(), msg); err != nil {
		t.Fatalf("second send: %v", err)
	}

	if len(creditSvc.debits) != 1 {
		t.Fatalf("expected one debit, got %d", len(creditSvc.debits))
	}
	if len(client.Messages()) != 1 {
		t.Fatalf("expected one provider send, got %d", len(client.Messages()))
	}
	if outboxStore.debited != 1 || outboxStore.sent != 1 {
		t.Fatalf("unexpected outbox transitions: debited=%d sent=%d", outboxStore.debited, outboxStore.sent)
	}
	if len(outboxStore.created) != 2 || outboxStore.created[0].Reference != msg.Reference {
		t.Fatalf("expected outbox reference lookup, got %+v", outboxStore.created)
	}
}

func TestServiceMarksOutboxRefundedOnProviderFailure(t *testing.T) {
	userID := uuid.New()
	creditSvc := &fakeCredits{}
	outboxStore := &fakeOutbox{}
	service := sms.NewService(
		creditSvc,
		routing.NewService(),
		map[string]provider.Provider{routing.ProviderMock: mock.NewProvider(mock.NewClient(true))},
		sms.Config{Outbox: outboxStore},
	)

	err := service.Send(context.Background(), sms.Message{UserID: userID, To: "+2348012345678", Content: "hello", Reference: "dunning:attempt-2"})
	if err == nil {
		t.Fatal("expected provider error")
	}

	if len(creditSvc.debits) != 1 || len(creditSvc.refunds) != 1 {
		t.Fatalf("expected one debit and one refund, got debits=%d refunds=%d", len(creditSvc.debits), len(creditSvc.refunds))
	}
	if outboxStore.debited != 1 || outboxStore.refunded != 1 {
		t.Fatalf("unexpected outbox transitions: debited=%d refunded=%d", outboxStore.debited, outboxStore.refunded)
	}
}
