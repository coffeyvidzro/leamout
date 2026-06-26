package webhook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

const DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

var ErrProcessorUnavailable = errors.New("payment webhook processor unavailable")

// Processor is implemented by the payment/domain layer.
//
// Keep database writes, idempotency checks, status reconciliation, invoice updates,
// and subscription renewal outside this HTTP package. The webhook handler only
// normalizes provider events and passes them forward.
type Processor interface {
	ProcessWebhookEvent(ctx context.Context, event *provider.WebhookEvent) error
}

type ProcessorFunc func(ctx context.Context, event *provider.WebhookEvent) error

func (fn ProcessorFunc) ProcessWebhookEvent(ctx context.Context, event *provider.WebhookEvent) error {
	return fn(ctx, event)
}

type HandlerConfig struct {
	Logger *slog.Logger

	// MaxBodyBytes protects the API from huge webhook payloads.
	// If zero or negative, DefaultMaxBodyBytes is used.
	MaxBodyBytes int64

	// ProcessingTimeout bounds downstream processing. If zero, the request context
	// is used without adding a timeout.
	ProcessingTimeout time.Duration

	// ReturnEventDetails can be enabled in local development. In production, keep
	// this false so provider callbacks do not receive internal details.
	ReturnEventDetails bool
}

type Handler struct {
	registry  *Registry
	processor Processor
	logger    *slog.Logger
	config    HandlerConfig
}

func NewHandler(registry *Registry, processor Processor, config HandlerConfig) *Handler {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = DefaultMaxBodyBytes
	}

	return &Handler{
		registry:  registry,
		processor: processor,
		logger:    config.Logger,
		config:    config,
	}
}

// RegisterRoutes mounts provider-neutral payment webhook routes.
//
// Expected mount:
//
//	paymentWebhooks := router.Group("/webhooks/payments")
//	handler.RegisterRoutes(paymentWebhooks)
//
// Resulting URLs:
//
//	POST /webhooks/payments/moolre
//	POST /webhooks/payments/pawapay
func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/:provider", h.Handle)
}

// Handle accepts payment webhooks at POST /webhooks/payments/:provider.
func (h *Handler) Handle(c *gin.Context) {
	if h.registry == nil {
		h.respondError(c, http.StatusInternalServerError, "webhook registry unavailable", nil)
		return
	}
	if h.processor == nil {
		h.respondError(c, http.StatusInternalServerError, "webhook processor unavailable", ErrProcessorUnavailable)
		return
	}

	providerID := h.providerIDFromRequest(c)
	if providerID == "" {
		h.respondError(c, http.StatusBadRequest, "missing provider", nil)
		return
	}

	paymentProvider, err := h.registry.MustGet(providerID)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "unknown payment provider", err)
		return
	}

	body, err := h.readBody(c)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.respondError(c, http.StatusRequestEntityTooLarge, "webhook body too large", err)
			return
		}
		h.respondError(c, http.StatusBadRequest, "invalid webhook body", err)
		return
	}

	request := provider.WebhookRequest{
		Headers: c.Request.Header.Clone(),
		Body:    body,
	}

	ctx := c.Request.Context()
	if h.config.ProcessingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.ProcessingTimeout)
		defer cancel()
	}

	verified := false
	if verifier, ok := paymentProvider.(provider.WebhookVerifier); ok {
		if err := verifier.VerifyWebhookSignature(ctx, request); err != nil {
			h.respondError(c, http.StatusUnauthorized, "invalid webhook signature", err)
			return
		}
		verified = true
	}

	event, err := paymentProvider.ParseWebhook(ctx, request)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid webhook payload", err)
		return
	}
	if event == nil {
		h.respondError(c, http.StatusBadRequest, "empty webhook event", nil)
		return
	}

	h.normalizeEvent(event, providerID, body, verified)

	if err := h.processor.ProcessWebhookEvent(ctx, event); err != nil {
		// Return 5xx so providers that support retries can retry the callback.
		h.respondError(c, http.StatusInternalServerError, "webhook processing failed", err)
		return
	}

	payload := gin.H{
		"success": true,
		"message": "webhook accepted",
	}
	if h.config.ReturnEventDetails {
		payload["provider"] = event.ProviderID
		payload["event_type"] = event.EventType
		payload["status"] = event.Status
		payload["external_ref"] = event.ExternalRef
		payload["provider_reference"] = event.ProviderReference
		payload["verified"] = event.Verified
	}

	c.JSON(http.StatusOK, payload)
}

func (h *Handler) readBody(c *gin.Context) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.config.MaxBodyBytes)
	defer func() { _ = c.Request.Body.Close() }()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, errors.New("empty webhook body")
	}

	return body, nil
}

func (h *Handler) providerIDFromRequest(c *gin.Context) provider.ID {
	raw := c.Param("provider")
	if raw == "" {
		raw = c.Param("provider_id")
	}
	if raw == "" {
		raw = c.Param("id")
	}

	return NormalizeProviderID(provider.ID(raw))
}

func (h *Handler) normalizeEvent(event *provider.WebhookEvent, providerID provider.ID, body []byte, verified bool) {
	if event.ProviderID == "" {
		event.ProviderID = providerID
	} else {
		event.ProviderID = NormalizeProviderID(event.ProviderID)
	}

	if event.Payload == nil {
		event.Payload = body
	}

	if verified {
		event.Verified = true
	}

	if event.Metadata == nil {
		event.Metadata = map[string]string{}
	}
}

func (h *Handler) respondError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		h.logger.WarnContext(c.Request.Context(),
			"payment webhook request failed",
			"status", status,
			"message", message,
			"error", err,
			"provider", c.Param("provider"),
		)
	}

	payload := gin.H{
		"success": false,
		"message": message,
	}

	c.JSON(status, payload)
}
