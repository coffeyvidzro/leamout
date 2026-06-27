package paymentmethod

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	items := h.service.List(ListParams{
		Country:  c.Query("country"),
		Currency: c.Query("currency"),
		Method:   c.Query("method"),
		Status:   c.Query("status"),
	})

	c.JSON(http.StatusOK, items)
}
