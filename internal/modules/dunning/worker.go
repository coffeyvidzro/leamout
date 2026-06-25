package dunning

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

type SendRenewalSMSArgs struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (SendRenewalSMSArgs) Kind() string { return "dunning_send_renewal_sms" }

type SendRenewalSMSWorker struct {
	river.WorkerDefaults[SendRenewalSMSArgs]

	service      *Service
	repository   *Repository
	shortBaseURL string
	log          *slog.Logger
}

func NewSendRenewalSMSWorker(service *Service, repository *Repository, shortBaseURL string, log *slog.Logger) *SendRenewalSMSWorker {
	return &SendRenewalSMSWorker{
		service:      service,
		repository:   repository,
		shortBaseURL: strings.TrimRight(shortBaseURL, "/"),
		log:          log,
	}
}

func (w *SendRenewalSMSWorker) Work(ctx context.Context, job *river.Job[SendRenewalSMSArgs]) error {
	attempt, err := w.service.Get(ctx, job.Args.UserID, job.Args.AttemptID)
	if err != nil {
		return fmt.Errorf("get dunning attempt: %w", err)
	}

	rawToken, _, err := w.service.CreateToken(ctx, attempt)
	if errors.Is(err, ErrActiveTokenExists) {
		return w.service.MarkAttemptSent(ctx, attempt.ID)
	}
	if err != nil {
		return fmt.Errorf("create dunning token: %w", err)
	}

	details, err := w.repository.GetNotificationDetails(ctx, attempt.UserID, attempt.ID)
	if err != nil {
		return fmt.Errorf("get dunning notification details: %w", err)
	}

	link := w.recoveryLink(rawToken)
	message := fmt.Sprintf("Your Leamout subscription expires soon. Renew here: %s", link)
	if w.log != nil {
		w.log.Info("mock renewal sms", slog.String("to", details.Phone), slog.String("message", message))
	} else {
		fmt.Printf("[MOCK SMS] To %s: %s\n", details.Phone, message)
	}

	return w.service.MarkAttemptSent(ctx, attempt.ID)
}

func (w *SendRenewalSMSWorker) recoveryLink(rawToken string) string {
	base := w.shortBaseURL
	if base == "" {
		base = "https://leam.out"
	}

	return base + "/r/" + url.PathEscape(rawToken)
}

type NotificationDetails struct {
	Phone string
}
