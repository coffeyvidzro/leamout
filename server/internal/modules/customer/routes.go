package customer

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	customers := router.Group("/customers")
	customers.Use(authMiddleware)

	customers.POST("", handler.Create)
	customers.GET("", handler.List)

	customers.GET("/external/:external_id/state", handler.GetStateByExternalID)
	customers.GET("/external/:external_id", handler.GetByExternalID)
	customers.PATCH("/external/:external_id", handler.UpdateByExternalID)
	customers.DELETE("/external/:external_id", handler.DeleteByExternalID)

	customers.GET("/:id/state", handler.GetState)
	customers.GET("/:id", handler.Get)
	customers.PATCH("/:id", handler.Update)
	customers.DELETE("/:id", handler.Delete)
}
