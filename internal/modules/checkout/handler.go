package checkout

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Show(c *gin.Context) {
	session, err := h.service.StartFromToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondCheckoutError(c, err)
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderCheckoutPage(c.Param("token"), session)))
}

func (h *Handler) MockPay(c *gin.Context) {
	session, err := h.service.CompleteMockPayment(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondCheckoutError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "mock payment successful",
		"session": session,
	})
}

func respondCheckoutError(c *gin.Context, err error) {
	if errors.Is(err, ErrInvalidToken) || errors.Is(err, dunning.ErrNotFound) || errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "checkout session not found"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "checkout failed"})
}

func renderCheckoutPage(rawToken string, session *Session) string {
	action := "/r/" + template.HTMLEscapeString(rawToken) + "/pay"
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Renew subscription</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 560px; margin: 48px auto; padding: 0 16px; color: #111827; }
    .card { border: 1px solid #e5e7eb; border-radius: 16px; padding: 24px; box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08); }
    .amount { font-size: 32px; font-weight: 700; margin: 16px 0; }
    button { background: #111827; color: white; border: 0; border-radius: 999px; padding: 12px 20px; font-weight: 700; cursor: pointer; }
  </style>
</head>
<body>
  <main class="card">
    <p>Leamout renewal checkout</p>
    <h1>Renew your subscription</h1>
    <div class="amount">%s %d</div>
    <p>Session ID: %s</p>
    <form method="post" action="%s">
      <button type="submit">Mock Pay</button>
    </form>
  </main>
</body>
</html>`, template.HTMLEscapeString(session.Currency), session.Amount, session.ID.String(), action)
}
