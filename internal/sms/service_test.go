package sms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/sms"
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
	if creditSvc.debits[0].Destination != "+234" {
		t.Fatalf("expected country-code ledger destination, got %q", creditSvc.debits[0].Destination)
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
	if creditSvc.debits[0].Destination != "+234" || creditSvc.refunds[0].Destination != "+234" {
		t.Fatalf("expected country-code ledger destinations, got debit=%q refund=%q", creditSvc.debits[0].Destination, creditSvc.refunds[0].Destination)
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
