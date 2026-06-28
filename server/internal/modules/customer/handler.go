package customer

import (
	"errors"
	"net/http"

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

	customer, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	customers, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customers"})
		return
	}

	c.JSON(http.StatusOK, customers)
}

func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	customer, err := h.service.Get(c.Request.Context(), userID, id)
	respondCustomer(c, customer, err)
}

func (h *Handler) GetState(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	state, err := h.service.GetState(c.Request.Context(), userID, id)
	respondState(c, state, err)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.Update(c.Request.Context(), userID, id, req)
	respondCustomer(c, customer, err)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, id); errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetByExternalID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	externalID := c.Param("external_id")

	customer, err := h.service.GetByExternalID(c.Request.Context(), userID, externalID)
	respondCustomer(c, customer, err)
}

func (h *Handler) GetStateByExternalID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	externalID := c.Param("external_id")

	state, err := h.service.GetStateByExternalID(c.Request.Context(), userID, externalID)
	respondState(c, state, err)
}

func (h *Handler) UpdateByExternalID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	externalID := c.Param("external_id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.UpdateByExternalID(c.Request.Context(), userID, externalID, req)
	respondCustomer(c, customer, err)
}

func (h *Handler) DeleteByExternalID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	externalID := c.Param("external_id")

	if err := h.service.DeleteByExternalID(c.Request.Context(), userID, externalID); errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer"})
		return
	}

	c.Status(http.StatusNoContent)
}

func respondCustomer(c *gin.Context, customer *Customer, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func respondState(c *gin.Context, state *State, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer state"})
		return
	}

	c.JSON(http.StatusOK, state)
}
