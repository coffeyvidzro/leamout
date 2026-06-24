package subscription

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	subscriptions := router.Group("/subscriptions")
	subscriptions.Use(authMiddleware)

	subscriptions.POST("", handler.Create)
	subscriptions.GET("", handler.List)
	subscriptions.GET("/:id", handler.Get)
	subscriptions.PATCH("/:id", handler.Update)
	subscriptions.DELETE("/:id", handler.Delete)
}
