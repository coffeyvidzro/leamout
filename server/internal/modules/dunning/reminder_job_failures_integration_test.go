package dunning_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	dunning "github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/sms"
	dunningworkflow "github.com/cuffeyvidzro/leamout/internal/workflows/dunning"
	"github.com/riverqueue/river"
)

type failingReminderSMSSender struct {
	err error
}

func (s failingReminderSMSSender) Send(context.Context, sms.Message) error {
	return s.err
}

func TestSendReminderWorkerRecordsRetryVisibilityAndCancelsAfterThreshold(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	worker := dunningworkflow.NewSendReminderWorker(fixture.Service, failingReminderSMSSender{err: errors.New("provider timeout")}, "https://lmt.test", nil)
	job := &river.Job[dunningworkflow.SendReminderArgs]{Args: dunningworkflow.SendReminderArgs{
		UserID:           fixture.UserID,
		SubscriptionID:   fixture.SubscriptionID,
		CustomerID:       fixture.CustomerID,
		CurrentPeriodEnd: fixture.Attempt.PeriodEnd,
	}}

	for run := 1; run <= dunningworkflow.MaxReminderJobFailures; run++ {
		if err := worker.Work(context.Background(), job); err == nil {
			t.Fatalf("expected reminder worker failure on run %d", run)
		}
	}

	failures, err := fixture.Service.ListReminderJobFailures(context.Background(), fixture.UserID)
	if err != nil {
		t.Fatalf("list reminder job failures: %v", err)
	}
	if len(failures) != dunningworkflow.MaxReminderJobFailures {
		t.Fatalf("expected %d reminder job failures, got %d", dunningworkflow.MaxReminderJobFailures, len(failures))
	}

	latest := failures[0]
	if latest.FailureNumber != dunningworkflow.MaxReminderJobFailures {
		t.Fatalf("expected latest failure number %d, got %d", dunningworkflow.MaxReminderJobFailures, latest.FailureNumber)
	}
	if latest.Status != dunning.ReminderJobFailureStatusRetryExhausted {
		t.Fatalf("expected latest failure status %s, got %s", dunning.ReminderJobFailureStatusRetryExhausted, latest.Status)
	}
	if !latest.Retryable {
		t.Fatal("expected provider timeout failure to be retryable")
	}
	if latest.ErrorType != dunningworkflow.ErrorTypeSMSSend {
		t.Fatalf("expected error type %s, got %s", dunningworkflow.ErrorTypeSMSSend, latest.ErrorType)
	}
	if latest.AttemptID == nil || *latest.AttemptID != fixture.Attempt.ID {
		t.Fatalf("expected failure attempt id %s, got %v", fixture.Attempt.ID, latest.AttemptID)
	}

	oldest := failures[len(failures)-1]
	if oldest.FailureNumber != 1 {
		t.Fatalf("expected first failure number 1, got %d", oldest.FailureNumber)
	}
	if oldest.Status != dunning.ReminderJobFailureStatusRetryScheduled {
		t.Fatalf("expected first failure status %s, got %s", dunning.ReminderJobFailureStatusRetryScheduled, oldest.Status)
	}

	attempt, err := fixture.Service.Get(context.Background(), fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("get dunning attempt: %v", err)
	}
	if attempt.Status != dunning.AttemptStatusCanceled {
		t.Fatalf("expected dunning attempt canceled after retry exhaustion, got %s", attempt.Status)
	}
	if attempt.CanceledAt == nil {
		t.Fatal("expected canceled_at after retry exhaustion")
	}
}

func TestSendReminderWorkerRecordsNonRetryableFailureImmediately(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	worker := dunningworkflow.NewSendReminderWorker(fixture.Service, failingReminderSMSSender{err: credits.ErrInsufficientBalance}, "https://lmt.test", nil)
	job := &river.Job[dunningworkflow.SendReminderArgs]{Args: dunningworkflow.SendReminderArgs{
		UserID:           fixture.UserID,
		SubscriptionID:   fixture.SubscriptionID,
		CustomerID:       fixture.CustomerID,
		CurrentPeriodEnd: fixture.Attempt.PeriodEnd,
	}}

	if err := worker.Work(context.Background(), job); err == nil {
		t.Fatal("expected reminder worker failure")
	}

	failures, err := fixture.Service.ListReminderJobFailures(context.Background(), fixture.UserID)
	if err != nil {
		t.Fatalf("list reminder job failures: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected one reminder job failure, got %d", len(failures))
	}

	failure := failures[0]
	if failure.FailureNumber != 1 {
		t.Fatalf("expected failure number 1, got %d", failure.FailureNumber)
	}
	if failure.Status != dunning.ReminderJobFailureStatusRetryExhausted {
		t.Fatalf("expected failure status %s, got %s", dunning.ReminderJobFailureStatusRetryExhausted, failure.Status)
	}
	if failure.Retryable {
		t.Fatal("expected insufficient credits failure to be non-retryable")
	}
	if failure.ErrorType != dunningworkflow.ErrorTypeInsufficientFunds {
		t.Fatalf("expected error type %s, got %s", dunningworkflow.ErrorTypeInsufficientFunds, failure.ErrorType)
	}

	attempt, err := fixture.Service.Get(context.Background(), fixture.UserID, fixture.Attempt.ID)
	if err != nil {
		t.Fatalf("get dunning attempt: %v", err)
	}
	if attempt.Status != dunning.AttemptStatusCanceled {
		t.Fatalf("expected dunning attempt canceled after non-retryable failure, got %s", attempt.Status)
	}
}
