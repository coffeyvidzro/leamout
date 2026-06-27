package checkout

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.Create(c.Request.Context(), userID, req)
	if errors.Is(err, ErrInvalidCheckoutRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": checkoutErrorMessage(err)})
		return
	}
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create checkout session", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	sessions, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to list checkout sessions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch checkout sessions"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkout session id"})
		return
	}

	session, err := h.service.Get(c.Request.Context(), userID, id)
	respondCheckout(c, session, err)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkout session id"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.Update(c.Request.Context(), userID, id, req)
	respondCheckout(c, session, err)
}

func (h *Handler) GetPublic(c *gin.Context) {
	session, err := h.service.GetPublic(c.Request.Context(), c.Param("clientSecret"))
	respondCheckout(c, session, err)
}

func (h *Handler) Pay(c *gin.Context) {
	var req PayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Intelligence = requestIntelligence(c)

	response, err := h.service.Pay(c.Request.Context(), c.Param("clientSecret"), req)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "checkout session not found"})
		return
	}
	if errors.Is(err, ErrInvalidCheckoutRequest) || errors.Is(err, ErrInvalidPaymentRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": checkoutErrorMessage(err)})
		return
	}
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "checkout payment failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start payment"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func respondCheckout(c *gin.Context, session *Session, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "checkout session not found"})
		return
	}
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to fetch checkout session", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch checkout session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func requestIntelligence(c *gin.Context) RequestIntelligence {
	info := RequestIntelligence{ClientIP: c.ClientIP()}
	if geo, ok := middleware.GetGeolocation(c); ok && geo != nil {
		info.DetectedCountry = geo.CountryCode
		info.DetectedSource = geo.Source
	}
	return info
}

func checkoutErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	for _, prefix := range []string{
		ErrInvalidCheckoutRequest.Error() + ":",
		ErrInvalidPaymentRequest.Error() + ":",
	} {
		message = strings.TrimSpace(strings.TrimPrefix(message, prefix))
	}
	if message == "" || message == ErrInvalidCheckoutRequest.Error() || message == ErrInvalidPaymentRequest.Error() {
		return "invalid checkout request"
	}
	return message
}
