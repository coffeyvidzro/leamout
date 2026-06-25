package pat

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	tokens := router.Group("/v1/personal-access-tokens")
	tokens.Use(authMiddleware)

	tokens.GET("", handler.List)
	tokens.POST("", handler.Create)
	tokens.DELETE("/:id", handler.Revoke)
}
