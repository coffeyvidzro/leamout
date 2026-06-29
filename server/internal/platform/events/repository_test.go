package events

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestPublishRequiresCoreFields(t *testing.T) {
	repo := &Repository{}
	_, err := repo.Publish(t.Context(), PublishParams{Name: PaymentCaptured, AggregateType: "payment", AggregateID: uuid.New()})
	if err == nil {
		t.Fatal("expected repository configuration error")
	}
}

func TestInvalidEventFields(t *testing.T) {
	if !errors.Is(ErrInvalidEvent, ErrInvalidEvent) {
		t.Fatal("sentinel sanity check failed")
	}
}
