package meter

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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

	meter, err := h.service.Create(c.Request.Context(), userID, req)
	respondMeter(c, meter, err, http.StatusCreated)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	params := ListParams{
		UserID:          userID,
		IncludeArchived: parseBool(c.Query("include_archived")),
		Page:            parseInt(c.Query("page"), 1),
		Limit:           parseInt(c.Query("limit"), 10),
	}

	meters, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list meters"})
		return
	}

	c.JSON(http.StatusOK, meters)
}

func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	meter, err := h.service.Get(c.Request.Context(), userID, id)
	respondMeter(c, meter, err, http.StatusOK)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	meter, err := h.service.Update(c.Request.Context(), userID, id, req)
	respondMeter(c, meter, err, http.StatusOK)
}

func (h *Handler) GetQuantities(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	start, err := parseOptionalTime(c.Query("start_timestamp"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_timestamp"})
		return
	}
	end, err := parseOptionalTime(c.Query("end_timestamp"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_timestamp"})
		return
	}

	quantity, err := h.service.GetQuantities(c.Request.Context(), QuantityParams{
		UserID:         userID,
		MeterID:        id,
		StartTimestamp: start,
		EndTimestamp:   end,
	})
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "meter not found"})
		return
	}
	if errors.Is(err, ErrInvalidMeter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get meter quantities"})
		return
	}

	c.JSON(http.StatusOK, quantity)
}

func respondMeter(c *gin.Context, meter *Meter, err error, successStatus int) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "meter not found"})
		return
	}
	if errors.Is(err, ErrInvalidMeter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process meter"})
		return
	}

	c.JSON(successStatus, meter)
}

func parseIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meter id"})
		return uuid.Nil, false
	}

	return id, true
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}

	parsed = parsed.UTC()
	return &parsed, nil
}
