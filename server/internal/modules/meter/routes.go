package meter

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	meters := router.Group("/meters")
	meters.Use(authMiddleware)

	meters.POST("", handler.Create)
	meters.GET("", handler.List)
	meters.GET("/:id", handler.Get)
	meters.PATCH("/:id", handler.Update)
	meters.GET("/:id/quantities", handler.GetQuantities)
}
