package dunning

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const SendReminderJobKind = "dunning_send_reminder"

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

func RegisterSendReminderWorker(workers *river.Workers, service *Service, sender provider.Provider, baseURL string, log *slog.Logger) {
	river.AddWorker(workers, NewSendReminderWorker(service, sender, baseURL, log))
}

type SendReminderWorker struct {
	river.WorkerDefaults[SendReminderArgs]

	service *Service
	sender  provider.Provider
	baseURL string
	log     *slog.Logger
}

func NewSendReminderWorker(service *Service, sender provider.Provider, baseURL string, log *slog.Logger) *SendReminderWorker {
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
	attempt, err := w.service.CreateOrReuseAttempt(ctx, CreateAttemptParams{
		UserID:         job.Args.UserID,
		SubscriptionID: job.Args.SubscriptionID,
		CustomerID:     &customerID,
		Reason:         AttemptReasonRenewalDue,
		PeriodEnd:      job.Args.CurrentPeriodEnd,
		Metadata: map[string]any{
			"source": "dunning_scanner",
		},
	})
	if err != nil {
		return fmt.Errorf("create dunning attempt: %w", err)
	}

	rawToken, _, err := w.service.CreateToken(ctx, attempt)
	if err != nil {
		if err == ErrActiveTokenExists {
			if err := w.service.RevokeAttemptTokens(ctx, attempt.UserID, attempt.ID); err != nil {
				return fmt.Errorf("revoke stale dunning tokens: %w", err)
			}

			rawToken, _, err = w.service.CreateToken(ctx, attempt)
			if err != nil {
				return fmt.Errorf("create replacement dunning token: %w", err)
			}
		} else {
			return fmt.Errorf("create dunning token: %w", err)
		}
	}

	details, err := w.service.GetReminderDetails(ctx, job.Args.UserID, job.Args.SubscriptionID)
	if err != nil {
		return fmt.Errorf("get dunning reminder details: %w", err)
	}

	link := w.recoveryLink(rawToken)
	message := fmt.Sprintf("Your Leamout subscription expires soon. Renew here: %s", link)
	if err := w.sender.Send(ctx, provider.Message{
		To:      details.CustomerPhone,
		From:    "Leamout",
		Content: message,
	}); err != nil {
		return fmt.Errorf("send dunning sms: %w", err)
	}

	if err := w.service.MarkAttemptSent(ctx, attempt.ID); err != nil {
		return fmt.Errorf("mark dunning attempt sent: %w", err)
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

func (w *SendReminderWorker) recoveryLink(rawToken string) string {
	if w.baseURL == "" {
		return "/dunning/" + url.PathEscape(rawToken)
	}

	return w.baseURL + "/dunning/" + url.PathEscape(rawToken)
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
