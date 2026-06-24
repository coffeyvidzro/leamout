package checkout

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	checkouts := router.Group("/checkouts")
	checkouts.Use(authMiddleware)

	checkouts.GET("", handler.List)
	checkouts.POST("", handler.Create)
	checkouts.GET("/:id", handler.Get)
	checkouts.PATCH("/:id", handler.Update)
}
