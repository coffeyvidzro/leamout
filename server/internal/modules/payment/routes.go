package payment

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	payments := router.Group("/payments")
	payments.Use(authMiddleware)

	payments.GET("", handler.List)
	payments.GET("/:id", handler.Get)
}
