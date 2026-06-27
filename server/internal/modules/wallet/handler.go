package wallet

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(c *gin.Context) {
	item, err := h.service.Get(c.Request.Context(), middleware.GetUserID(c), c.Param("currency"))
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallet"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListLedger(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.service.ListLedger(c.Request.Context(), ListLedgerParams{
		UserID:   middleware.GetUserID(c),
		Currency: c.Param("currency"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallet ledger"})
		return
	}

	c.JSON(http.StatusOK, items)
}
