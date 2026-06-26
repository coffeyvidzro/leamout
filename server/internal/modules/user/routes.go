package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	users := router.Group("/users")
	users.Use(authMiddleware)

	users.GET("/me", handler.GetMe)
	users.PATCH("/me", handler.UpdateMe)
	users.DELETE("/me", handler.DeleteMe)
}
