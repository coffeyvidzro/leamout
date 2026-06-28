package benefit

import (
	"errors"
	"net/http"
	"strconv"

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

	benefit, err := h.service.Create(c.Request.Context(), userID, req)
	respondBenefit(c, benefit, err, http.StatusCreated)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	benefits, err := h.service.List(c.Request.Context(), ListParams{
		UserID:          userID,
		Type:            Type(c.Query("type")),
		IncludeArchived: parseBool(c.Query("include_archived")),
		Page:            parseInt(c.Query("page"), 1),
		Limit:           parseInt(c.Query("limit"), 10),
	})
	if errors.Is(err, ErrInvalidBenefit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list benefits"})
		return
	}

	c.JSON(http.StatusOK, benefits)
}

func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	benefit, err := h.service.Get(c.Request.Context(), userID, id)
	respondBenefit(c, benefit, err, http.StatusOK)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	benefit, err := h.service.Update(c.Request.Context(), userID, id, req)
	respondBenefit(c, benefit, err, http.StatusOK)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.service.Delete(c.Request.Context(), userID, id)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "benefit not found"})
		return
	}
	if errors.Is(err, ErrInvalidBenefit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to archive benefit"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListGrants(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	customerID, ok := parseOptionalUUID(c, "customer_id")
	if !ok {
		return
	}

	grants, err := h.service.ListGrants(c.Request.Context(), ListGrantsParams{
		UserID:     userID,
		BenefitID:  id,
		CustomerID: customerID,
		Status:     GrantStatus(c.Query("status")),
		Page:       parseInt(c.Query("page"), 1),
		Limit:      parseInt(c.Query("limit"), 10),
	})
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "benefit not found"})
		return
	}
	if errors.Is(err, ErrInvalidBenefit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list benefit grants"})
		return
	}

	c.JSON(http.StatusOK, grants)
}

func respondBenefit(c *gin.Context, benefit *Benefit, err error, successStatus int) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "benefit not found"})
		return
	}
	if errors.Is(err, ErrInvalidBenefit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process benefit"})
		return
	}

	c.JSON(successStatus, benefit)
}

func parseIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benefit id"})
		return uuid.Nil, false
	}

	return id, true
}

func parseOptionalUUID(c *gin.Context, name string) (*uuid.UUID, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return nil, false
	}

	return &id, true
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
