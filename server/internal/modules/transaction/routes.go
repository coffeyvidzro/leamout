package transaction

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	transactions := router.Group("/transactions")
	transactions.Use(authMiddleware)

	transactions.GET("", handler.List)
	transactions.GET("/:id", handler.Get)
}
