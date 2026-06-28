package customermeter

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	customerMeters := router.Group("/customer-meters")
	customerMeters.Use(authMiddleware)

	customerMeters.GET("", handler.List)
	customerMeters.GET("/:id", handler.Get)
}
