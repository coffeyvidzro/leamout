package usageevent

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	events := router.Group("/events")
	events.Use(authMiddleware)

	events.POST("/ingest", handler.Ingest)
	events.GET("", handler.List)
	events.GET("/:id", handler.Get)
}
