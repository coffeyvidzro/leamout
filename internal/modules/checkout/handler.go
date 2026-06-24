package checkout

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

	session, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	sessions, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
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

func respondCheckout(c *gin.Context, session *Session, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "checkout session not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch checkout session"})
		return
	}

	c.JSON(http.StatusOK, session)
}
