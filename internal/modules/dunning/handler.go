package dunning

import (
	"errors"
	"net/http"
	"net/url"

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

func (h *Handler) OpenRecoveryLink(c *gin.Context) {
	checkoutSession, err := h.service.OpenRecoveryLink(c.Request.Context(), c.Param("token"))
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidRecoveryLink) {
		c.JSON(http.StatusNotFound, gin.H{"error": "recovery link not found or expired"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open recovery link"})
		return
	}

	c.Redirect(http.StatusFound, "/checkout/"+url.PathEscape(checkoutSession.ClientSecret))
}

func (h *Handler) List(c *gin.Context) {
	attempts, err := h.service.List(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch dunning attempts"})
		return
	}

	c.JSON(http.StatusOK, attempts)
}

func (h *Handler) Get(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dunning attempt id"})
		return
	}

	attempt, err := h.service.Get(c.Request.Context(), middleware.GetUserID(c), attemptID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "dunning attempt not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch dunning attempt"})
		return
	}

	c.JSON(http.StatusOK, attempt)
}
