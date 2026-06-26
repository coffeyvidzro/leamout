package product

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	products := router.Group("/products")
	products.Use(authMiddleware)

	products.POST("", handler.Create)
	products.GET("", handler.List)
	products.GET("/:id", handler.Get)
	products.PATCH("/:id", handler.Update)
	products.DELETE("/:id", handler.Delete)
}
