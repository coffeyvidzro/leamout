package tola

import (
	"context"
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/cuffeyvidzro/leamout/internal/payment/webhook"
)

type WebhookHandler struct {
	log *slog.Logger
}

func NewWebhookHandler(log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		log: log,
	}
}

func (h *WebhookHandler) Provider() payment.ProviderName {
	return payment.ProviderTola
}

func (h *WebhookHandler) HandleWebhook(ctx context.Context, event webhook.Event) error {
	// TODO: parse Tola callback payload and update transaction/payment status.
	if h.log != nil {
		h.log.InfoContext(ctx, "received tola webhook", "event_type", event.EventType)
	}

	return nil
}
