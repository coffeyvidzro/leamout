package wallet

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	wallets := router.Group("/wallets")
	wallets.Use(authMiddleware)

	wallets.GET("/:currency", handler.Get)
	wallets.GET("/:currency/ledger", handler.ListLedger)
}
