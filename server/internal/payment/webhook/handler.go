package webhook

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/gin-gonic/gin"
)

const maxWebhookBodyBytes = 1 << 20 // 1MB

type Handler struct {
	registry *Registry
}

func NewHandler(registry *Registry) *Handler {
	if registry == nil {
		registry = NewRegistry()
	}

	return &Handler{
		registry: registry,
	}
}

func (h *Handler) Handle(c *gin.Context) {
	providerName := payment.ProviderName(strings.ToLower(strings.TrimSpace(c.Param("provider"))))
	if providerName == "" {
		respondError(c, http.StatusBadRequest, "missing webhook provider")
		return
	}

	eventType := strings.ToLower(strings.TrimSpace(c.Param("eventType")))

	providerHandler, ok := h.registry.Get(providerName)
	if !ok {
		respondError(c, http.StatusNotFound, fmt.Sprintf("webhook provider %s is not registered", providerName))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes)

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid webhook body")
		return
	}

	event := Event{
		Provider:  providerName,
		EventType: eventType,
		RawBody:   rawBody,
		Headers:   c.Request.Header,
		Query:     c.Request.URL.Query(),
	}

	if err := providerHandler.HandleWebhook(c.Request.Context(), event); err != nil {
		status := http.StatusInternalServerError

		if errors.Is(err, ErrInvalidWebhookPayload) {
			status = http.StatusBadRequest
		}

		if errors.Is(err, ErrInvalidWebhookSignature) {
			status = http.StatusUnauthorized
		}

		respondError(c, status, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"received": true,
	})
}

var (
	ErrInvalidWebhookPayload   = errors.New("invalid webhook payload")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
)

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}
