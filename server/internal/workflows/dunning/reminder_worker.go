package dunning

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	dunningmodule "github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/sms"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const (
	SendReminderJobKind       = "dunning_send_reminder"
	maxReminderJobFailures    = 3
	errorTypeAttemptCreate     = "attempt_create_failed"
	errorTypeTokenCreate       = "token_create_failed"
	errorTypeReminderDetails   = "reminder_details_failed"
	errorTypeSMSSend           = "sms_send_failed"
	errorTypeInsufficientFunds = "insufficient_communication_credits"
	errorTypeMarkSent          = "mark_sent_failed"
)

type SendReminderArgs struct {
	UserID           uuid.UUID `json:"user_id" river:"unique"`
	SubscriptionID   uuid.UUID `json:"subscription_id" river:"unique"`
	CustomerID       uuid.UUID `json:"customer_id" river:"unique"`
	CurrentPeriodEnd time.Time `json:"current_period_end" river:"unique"`
}

func (SendReminderArgs) Kind() string { return SendReminderJobKind }

func (SendReminderArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	}
}

func RegisterReminderJobKind(workers *river.Workers) {
	river.AddWorker(workers, river.WorkFunc(func(context.Context, *river.Job[SendReminderArgs]) error {
		return fmt.Errorf("%s worker is not registered in this process", SendReminderJobKind)
	}))
}

type SMSSender interface {
	Send(context.Context, sms.Message) error
}

func RegisterSendReminderWorker(workers *river.Workers, service *dunningmodule.Service, sender SMSSender, baseURL string, log *slog.Logger) {
	river.AddWorker(workers, NewSendReminderWorker(service, sender, baseURL, log))
}

type SendReminderWorker struct {
	river.WorkerDefaults[SendReminderArgs]

	service *dunningmodule.Service
	sender  SMSSender
	baseURL string
	log     *slog.Logger
}

func NewSendReminderWorker(service *dunningmodule.Service, sender SMSSender, baseURL string, log *slog.Logger) *SendReminderWorker {
	return &SendReminderWorker{
		service: service,
		sender:  sender,
		baseURL: strings.TrimRight(baseURL, "/"),
		log:     log,
	}
}

func (w *SendReminderWorker) Work(ctx context.Context, job *river.Job[SendReminderArgs]) error {
	if w.service == nil {
		return fmt.Errorf("dunning service is not configured")
	}
	if w.sender == nil {
		return fmt.Errorf("sms sender is not configured")
	}

	customerID := job.Args.CustomerID
	attempt, err := w.service.CreateOrReuseAttempt(ctx, dunningmodule.CreateAttemptParams{
		UserID:         job.Args.UserID,
		SubscriptionID: job.Args.SubscriptionID,
		CustomerID:     &customerID,
		Reason:         dunningmodule.AttemptReasonRenewalDue,
		PeriodEnd:      job.Args.CurrentPeriodEnd,
		Metadata: map[string]any{
			"source": "dunning_scanner",
		},
	})
	if err != nil {
		return w.handleFailure(ctx, job.Args, nil, errorTypeAttemptCreate, fmt.Errorf("create dunning attempt: %w", err), true)
	}

	rawToken, _, err := w.service.CreateToken(ctx, attempt)
	if err != nil {
		if err == dunningmodule.ErrActiveTokenExists {
			if err := w.service.RevokeAttemptTokens(ctx, attempt.UserID, attempt.ID); err != nil {
				return w.handleFailure(ctx, job.Args, attempt, errorTypeTokenCreate, fmt.Errorf("revoke stale dunning tokens: %w", err), true)
			}

			rawToken, _, err = w.service.CreateToken(ctx, attempt)
			if err != nil {
				return w.handleFailure(ctx, job.Args, attempt, errorTypeTokenCreate, fmt.Errorf("create replacement dunning token: %w", err), true)
			}
		} else {
			return w.handleFailure(ctx, job.Args, attempt, errorTypeTokenCreate, fmt.Errorf("create dunning token: %w", err), true)
		}
	}

	details, err := w.service.GetReminderDetails(ctx, job.Args.UserID, job.Args.SubscriptionID)
	if err != nil {
		return w.handleFailure(ctx, job.Args, attempt, errorTypeReminderDetails, fmt.Errorf("get dunning reminder details: %w", err), true)
	}

	link := w.recoveryLink(rawToken)
	message := fmt.Sprintf("Your Leamout subscription expires soon. Renew here: %s", link)
	if err := w.sender.Send(ctx, sms.Message{
		UserID:    job.Args.UserID,
		To:        details.CustomerPhone,
		Content:   message,
		Reference: "dunning_sms:" + attempt.ID.String(),
		Metadata: map[string]any{
			"dunning_attempt_id": attempt.ID.String(),
			"subscription_id":    job.Args.SubscriptionID.String(),
		},
	}); err != nil {
		if errors.Is(err, credits.ErrInsufficientBalance) {
			return w.handleFailure(ctx, job.Args, attempt, errorTypeInsufficientFunds, fmt.Errorf("send dunning sms: %w", err), false)
		}
		return w.handleFailure(ctx, job.Args, attempt, errorTypeSMSSend, fmt.Errorf("send dunning sms: %w", err), true)
	}

	if err := w.service.MarkAttemptSent(ctx, attempt.ID); err != nil {
		return w.handleFailure(ctx, job.Args, attempt, errorTypeMarkSent, fmt.Errorf("mark dunning attempt sent: %w", err), true)
	}

	w.logInfo("sent dunning reminder", job.Args, attempt.ID)
	if w.log != nil {
		w.log.Info(
			"[MOCK SMS]",
			slog.String("to", details.CustomerPhone),
			slog.String("message", message),
		)
	}

	return nil
}

func (w *SendReminderWorker) handleFailure(ctx context.Context, args SendReminderArgs, attempt *dunningmodule.Attempt, errorType string, err error, retryable bool) error {
	status := dunningmodule.ReminderJobFailureStatusRetryScheduled
	if !retryable || w.nextFailureNumber(ctx, args) >= maxReminderJobFailures {
		status = dunningmodule.ReminderJobFailureStatusRetryExhausted
	}

	var attemptID *uuid.UUID
	if attempt != nil {
		attemptID = &attempt.ID
	}

	failure, recordErr := w.service.RecordReminderJobFailure(ctx, dunningmodule.RecordReminderJobFailureParams{
		UserID:           args.UserID,
		SubscriptionID:   args.SubscriptionID,
		CustomerID:       args.CustomerID,
		AttemptID:        attemptID,
		CurrentPeriodEnd: args.CurrentPeriodEnd,
		Status:           status,
		ErrorType:        errorType,
		ErrorMessage:     err.Error(),
		Retryable:        retryable,
		Metadata: map[string]any{
			"source":   "dunning_reminder_worker",
			"job_kind": SendReminderJobKind,
		},
	})
	if recordErr != nil {
		return fmt.Errorf("%w; failed to record dunning reminder job failure: %v", err, recordErr)
	}

	w.logFailure("dunning reminder job failed", args, attemptID, failure)
	if failure.Status == dunningmodule.ReminderJobFailureStatusRetryExhausted {
		if attempt != nil {
			if cancelErr := w.service.MarkAttemptCanceled(ctx, attempt.ID, map[string]any{
				"source":          "dunning_reminder_worker",
				"failure_id":      failure.ID.String(),
				"failure_number":  failure.FailureNumber,
				"error_type":      failure.ErrorType,
				"original_status": attempt.Status,
			}); cancelErr != nil && !errors.Is(cancelErr, dunningmodule.ErrInvalidDunningTransition) && !errors.Is(cancelErr, dunningmodule.ErrTransitionSkipped) {
				return fmt.Errorf("%w; failed to cancel dunning attempt after retry exhaustion: %v", err, cancelErr)
			}
		}
		return river.JobCancel(fmt.Errorf("stop dunning reminder after %d failed attempts: %w", failure.FailureNumber, err))
	}

	return err
}

func (w *SendReminderWorker) nextFailureNumber(ctx context.Context, args SendReminderArgs) int {
	failures, err := w.service.ListReminderJobFailures(ctx, args.UserID)
	if err != nil {
		return 1
	}

	next := 1
	for _, failure := range failures {
		if failure.SubscriptionID == args.SubscriptionID && failure.CustomerID == args.CustomerID && failure.CurrentPeriodEnd.Equal(args.CurrentPeriodEnd) {
			next++
		}
	}
	return next
}

func (w *SendReminderWorker) recoveryLink(rawToken string) string {
	if w.baseURL == "" {
		return "/r/" + url.PathEscape(rawToken)
	}

	return w.baseURL + "/r/" + url.PathEscape(rawToken)
}

func (w *SendReminderWorker) logInfo(message string, args SendReminderArgs, attemptID uuid.UUID) {
	if w.log == nil {
		return
	}

	w.log.Info(
		message,
		slog.String("user_id", args.UserID.String()),
		slog.String("subscription_id", args.SubscriptionID.String()),
		slog.String("customer_id", args.CustomerID.String()),
		slog.String("dunning_attempt_id", attemptID.String()),
	)
}

func (w *SendReminderWorker) logFailure(message string, args SendReminderArgs, attemptID *uuid.UUID, failure *dunningmodule.ReminderJobFailure) {
	if w.log == nil || failure == nil {
		return
	}

	attrs := []any{
		slog.String("user_id", args.UserID.String()),
		slog.String("subscription_id", args.SubscriptionID.String()),
		slog.String("customer_id", args.CustomerID.String()),
		slog.String("failure_id", failure.ID.String()),
		slog.Int("failure_number", failure.FailureNumber),
		slog.String("failure_status", string(failure.Status)),
		slog.String("error_type", failure.ErrorType),
	}
	if attemptID != nil {
		attrs = append(attrs, slog.String("dunning_attempt_id", attemptID.String()))
	}

	w.log.Warn(message, attrs...)
}
