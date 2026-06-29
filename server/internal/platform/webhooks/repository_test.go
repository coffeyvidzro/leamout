package webhooks

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestEnqueueRequiresConfiguredRepository(t *testing.T) {
	repo := &Repository{}
	_, err := repo.Enqueue(t.Context(), EnqueueParams{
		UserID:        uuid.New(),
		EventType:     EventPaymentCaptured,
		AggregateType: "payment",
		AggregateID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected repository configuration error")
	}
}

func TestErrInvalidWebhookSentinel(t *testing.T) {
	if !errors.Is(ErrInvalidWebhook, ErrInvalidWebhook) {
		t.Fatal("sentinel sanity check failed")
	}
}
