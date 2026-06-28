package benefit

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	benefits := router.Group("/benefits")
	benefits.Use(authMiddleware)

	benefits.POST("", handler.Create)
	benefits.GET("", handler.List)
	benefits.GET("/:id", handler.Get)
	benefits.PATCH("/:id", handler.Update)
	benefits.DELETE("/:id", handler.Delete)
	benefits.GET("/:id/grants", handler.ListGrants)
}
