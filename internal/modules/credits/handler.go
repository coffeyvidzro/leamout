package credits

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

type TopUpRequest struct {
	Amount      int64          `json:"amount" binding:"required,gt=0"`
	Reference   string         `json:"reference"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

func (h *Handler) GetBalance(c *gin.Context) {
	userID := middleware.GetUserID(c)

	balance, err := h.service.GetBalance(c.Request.Context(), userID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusOK, Balance{
			UserID:   userID,
			Balance:  0,
			Currency: CurrencyGHS,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch credit balance"})
		return
	}

	c.JSON(http.StatusOK, balance)
}

func (h *Handler) ListLedger(c *gin.Context) {
	userID := middleware.GetUserID(c)

	entries, err := h.service.ListLedger(c.Request.Context(), ListLedgerParams{
		UserID: userID,
		Limit:  queryInt(c, "limit", 100),
		Offset: queryInt(c, "offset", 0),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch credit ledger"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (h *Handler) TopUp(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance, err := h.service.TopUp(c.Request.Context(), TopUpParams{
		UserID:      userID,
		Amount:      req.Amount,
		Reference:   req.Reference,
		Description: req.Description,
		Metadata:    req.Metadata,
	})
	if errors.Is(err, ErrInvalidAmount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to top up communication credits"})
		return
	}

	c.JSON(http.StatusCreated, balance)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
