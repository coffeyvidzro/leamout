package customermeter

import (
	"errors"
	"net/http"
	"strconv"
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

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	params := ListParams{
		UserID:             userID,
		ExternalCustomerID: c.Query("external_customer_id"),
		Page:               intQuery(c, "page", 1),
		Limit:              intQuery(c, "limit", 10),
	}

	if raw := strings.TrimSpace(c.Query("customer_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
		params.CustomerID = &id
	}
	if raw := strings.TrimSpace(c.Query("meter_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meter_id"})
			return
		}
		params.MeterID = &id
	}

	response, err := h.service.List(c.Request.Context(), params)
	respondList(c, response, err)
}

func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer meter id"})
		return
	}

	meter, err := h.service.Get(c.Request.Context(), userID, id)
	respondCustomerMeter(c, meter, err)
}

func respondList(c *gin.Context, response *ListResponse, err error) {
	if errors.Is(err, ErrInvalidCustomerMeter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer meters"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func respondCustomerMeter(c *gin.Context, meter *CustomerMeter, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer meter not found"})
		return
	}
	if errors.Is(err, ErrInvalidCustomerMeter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer meter"})
		return
	}

	c.JSON(http.StatusOK, meter)
}

func intQuery(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
