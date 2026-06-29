package dunning

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDunningStatusTransitionsWriteHistory(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)

	if err := fixture.Service.MarkAttemptSent(context.Background(), fixture.Attempt.ID); err != nil {
		t.Fatalf("mark attempt sent: %v", err)
	}
	if err := fixture.Service.MarkAttemptPaid(context.Background(), fixture.Attempt.ID); err != nil {
		t.Fatalf("mark attempt paid: %v", err)
	}

	transitions, err := fixture.Service.ListAttemptTransitions(context.Background(), fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("list dunning attempt transitions: %v", err)
	}
	if len(transitions) != 2 {
		t.Fatalf("expected two dunning transitions, got %d", len(transitions))
	}

	assertDunningTransition(t, transitions[0], dunningTransitionActorWorker, dunningTransitionReasonReminderSent, AttemptStatusPending, AttemptStatusSent)
	if transitions[0].Metadata["source"] != "dunning_reminder_worker" {
		t.Fatalf("expected sent transition source dunning_reminder_worker, got %v", transitions[0].Metadata["source"])
	}

	assertDunningTransition(t, transitions[1], "system", dunningTransitionReasonRenewalPaid, AttemptStatusSent, AttemptStatusPaid)
	if transitions[1].Metadata["source"] != "dunning_service" {
		t.Fatalf("expected paid transition source dunning_service, got %v", transitions[1].Metadata["source"])
	}
}

func TestDunningSameStatusTransitionDoesNotDuplicateHistory(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)

	if err := fixture.Service.MarkAttemptSent(context.Background(), fixture.Attempt.ID); err != nil {
		t.Fatalf("mark attempt sent: %v", err)
	}
	if err := fixture.Service.MarkAttemptSent(context.Background(), fixture.Attempt.ID); err != nil {
		t.Fatalf("mark attempt sent again: %v", err)
	}

	transitions, err := fixture.Service.ListAttemptTransitions(context.Background(), fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("list dunning attempt transitions: %v", err)
	}
	if len(transitions) != 1 {
		t.Fatalf("expected one dunning transition after repeated sent call, got %d", len(transitions))
	}
	assertDunningTransition(t, transitions[0], dunningTransitionActorWorker, dunningTransitionReasonReminderSent, AttemptStatusPending, AttemptStatusSent)
}

func TestDunningDirectStatusUpdateWritesDefaultHistory(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)

	_, err := fixture.Pool.Exec(context.Background(), `
UPDATE dunning_attempts
SET status = 'paid',
	sent_at = COALESCE(sent_at, NOW()),
	paid_at = COALESCE(paid_at, NOW())
WHERE user_id = $1
  AND id = $2`, fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("directly mark attempt paid: %v", err)
	}

	transitions, err := fixture.Service.ListAttemptTransitions(context.Background(), fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("list dunning attempt transitions: %v", err)
	}
	if len(transitions) != 1 {
		t.Fatalf("expected one dunning transition after direct update, got %d", len(transitions))
	}
	assertDunningTransition(t, transitions[0], "system", "status_update", AttemptStatusPending, AttemptStatusPaid)
}

func assertDunningTransition(t *testing.T, transition AttemptTransition, actor, reason string, previous, next AttemptStatus) {
	t.Helper()

	if transition.ID == uuid.Nil {
		t.Fatal("expected transition id")
	}
	if transition.UserID == uuid.Nil {
		t.Fatal("expected transition user id")
	}
	if transition.AttemptID == uuid.Nil {
		t.Fatal("expected transition attempt id")
	}
	if transition.Actor != actor {
		t.Fatalf("expected actor %q, got %q", actor, transition.Actor)
	}
	if transition.Reason != reason {
		t.Fatalf("expected reason %q, got %q", reason, transition.Reason)
	}
	if transition.PreviousStatus != previous {
		t.Fatalf("expected previous status %q, got %q", previous, transition.PreviousStatus)
	}
	if transition.NextStatus != next {
		t.Fatalf("expected next status %q, got %q", next, transition.NextStatus)
	}
	if transition.CreatedAt.IsZero() {
		t.Fatal("expected transition created_at")
	}
}
