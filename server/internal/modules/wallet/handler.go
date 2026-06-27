package wallet

import (
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

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), ListWalletsParams{
		UserID:   middleware.GetUserID(c),
		Country:  c.Query("country"),
		Currency: c.Query("currency"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallets": items})
}

func (h *Handler) ListLedger(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.service.ListLedger(c.Request.Context(), ListLedgerParams{
		UserID:   middleware.GetUserID(c),
		Country:  c.Query("country"),
		Currency: c.Query("currency"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallet ledger"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ledger": items})
}
