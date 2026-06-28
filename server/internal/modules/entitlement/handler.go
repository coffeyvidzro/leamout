package entitlement

import (
	"errors"
	"net/http"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Check(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.Check(c.Request.Context(), userID, req)
	respondCheck(c, response, err)
}

func respondCheck(c *gin.Context, response *CheckResponse, err error) {
	if errors.Is(err, ErrInvalidCheck) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide exactly one of customer_id or external_customer_id, plus a benefit code"})
		return
	}
	if errors.Is(err, ErrCustomerNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check entitlement"})
		return
	}

	c.JSON(http.StatusOK, response)
}
