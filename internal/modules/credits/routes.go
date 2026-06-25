package credits

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	credits := router.Group("/credits")
	credits.Use(authMiddleware)

	// Creator credit balance
	credits.GET("", handler.GetBalance)

	// Creator credit ledger/history
	credits.GET("/ledger", handler.ListLedger)

	// Creator tops up communication balance
	credits.POST("/topup", handler.TopUp)
}
