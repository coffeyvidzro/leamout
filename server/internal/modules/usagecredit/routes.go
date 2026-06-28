package usagecredit

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	credits := router.Group("/usage-credits")
	credits.Use(authMiddleware)

	credits.GET("/grants", handler.ListGrants)
	credits.GET("/ledger", handler.ListLedger)
}
