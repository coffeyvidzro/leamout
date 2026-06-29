package usageevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestHandleUsageEventInsertOutcomeConsumesInsertedEvent(t *testing.T) {
	eventID := uuid.New()
	userID := uuid.New()
	response := &IngestResponse{}
	called := 0

	err := handleUsageEventInsertOutcome(
		context.Background(),
		nil,
		userID,
		response,
		&eventID,
		true,
		CreateParams{
			Timestamp: nowUTC(),
			Name:      "api_call",
			Source:    SourceUser,
			Metadata:  map[string]any{"quantity": 1},
		},
		func(ctx context.Context, tx pgx.Tx, receivedUserID, receivedEventID uuid.UUID, event CreateParams) error {
			called++
			if receivedUserID != userID {
				t.Fatalf("expected user id %s, got %s", userID, receivedUserID)
			}
			if receivedEventID != eventID {
				t.Fatalf("expected event id %s, got %s", eventID, receivedEventID)
			}
			if event.Name != "api_call" {
				t.Fatalf("expected event name api_call, got %q", event.Name)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called != 1 {
		t.Fatalf("expected consumer to be called once, got %d", called)
	}
	if response.Inserted != 1 {
		t.Fatalf("expected inserted count 1, got %d", response.Inserted)
	}
	if response.Duplicates != 0 {
		t.Fatalf("expected duplicate count 0, got %d", response.Duplicates)
	}
}

func TestHandleUsageEventInsertOutcomeSkipsConsumptionForDuplicateEvent(t *testing.T) {
	response := &IngestResponse{}
	consumerErr := errors.New("consumer should not be called for duplicate usage event")

	err := handleUsageEventInsertOutcome(
		context.Background(),
		nil,
		uuid.New(),
		response,
		nil,
		false,
		CreateParams{
			Timestamp: time.Now().UTC(),
			Name:      "api_call",
			Source:    SourceUser,
			Metadata:  map[string]any{"quantity": 1},
		},
		func(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, CreateParams) error {
			return consumerErr
		},
	)
	if err != nil {
		t.Fatalf("expected duplicate usage event to skip consumption, got %v", err)
	}
	if response.Inserted != 0 {
		t.Fatalf("expected inserted count 0, got %d", response.Inserted)
	}
	if response.Duplicates != 1 {
		t.Fatalf("expected duplicate count 1, got %d", response.Duplicates)
	}
}

func TestHandleUsageEventInsertOutcomeReturnsErrorWhenInsertedEventHasNoID(t *testing.T) {
	response := &IngestResponse{}

	err := handleUsageEventInsertOutcome(
		context.Background(),
		nil,
		uuid.New(),
		response,
		nil,
		true,
		CreateParams{Name: "api_call", Source: SourceUser},
		func(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, CreateParams) error {
			t.Fatal("consumer should not be called without an event id")
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if response.Inserted != 0 {
		t.Fatalf("expected inserted count 0, got %d", response.Inserted)
	}
	if response.Duplicates != 0 {
		t.Fatalf("expected duplicate count 0, got %d", response.Duplicates)
	}
}
