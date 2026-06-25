package pat

import (
	"errors"
	"net/http"

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
	tokens, err := h.service.List(c.Request.Context(), currentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch personal access tokens"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.Create(c.Request.Context(), currentUserID(c), req)
	if errors.Is(err, ErrInvalidExpiry) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create personal access token"})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *Handler) Revoke(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid personal access token id"})
		return
	}

	if err := h.service.Revoke(c.Request.Context(), currentUserID(c), tokenID); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "personal access token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke personal access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "personal access token revoked"})
}

func currentUserID(c *gin.Context) uuid.UUID {
	value, ok := c.Get("userID")
	if !ok {
		return uuid.Nil
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}

	return userID
}
