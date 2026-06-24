package sms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/sms"
	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
	"github.com/cuffeyvidzro/leamout/internal/sms/provider/mock"
)

func TestServiceAppliesDefaultSenderAndTrimsMessage(t *testing.T) {
	client := mock.NewClient(false)
	service := sms.NewService(mock.NewProvider(client), sms.Config{DefaultFrom: "Leamout"})

	err := service.Send(context.Background(), provider.Message{
		To:      " +233501234567 ",
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
		t.Fatalf("expected default sender, got %q", messages[0].Message.From)
	}
	if messages[0].Message.To != "+233501234567" {
		t.Fatalf("expected trimmed recipient, got %q", messages[0].Message.To)
	}
	if messages[0].Message.Content != "Renew here" {
		t.Fatalf("expected trimmed content, got %q", messages[0].Message.Content)
	}
}

func TestServiceValidatesRequiredFields(t *testing.T) {
	service := sms.NewService(mock.NewProvider(mock.NewClient(false)), sms.Config{DefaultFrom: "Leamout"})

	if err := service.Send(context.Background(), provider.Message{Content: "hello"}); !errors.Is(err, sms.ErrRecipientRequired) {
		t.Fatalf("expected recipient error, got %v", err)
	}
	if err := service.Send(context.Background(), provider.Message{To: "+233501234567"}); !errors.Is(err, sms.ErrContentRequired) {
		t.Fatalf("expected content error, got %v", err)
	}
}

func TestServiceRequiresProvider(t *testing.T) {
	service := sms.NewService(nil, sms.Config{})

	if err := service.Send(context.Background(), provider.Message{To: "+233501234567", Content: "hello"}); !errors.Is(err, sms.ErrProviderRequired) {
		t.Fatalf("expected provider error, got %v", err)
	}
}
