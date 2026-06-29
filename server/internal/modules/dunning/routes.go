package dunning

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	// Public system-generated recovery link. Example: https://leam.cc/r/<token>.
	router.GET("/dunning/:token", handler.OpenRecoveryLink)

	// Protected merchant visibility. Dunning attempts are system-created by jobs.
	dunning := router.Group("/dunning-events")
	dunning.Use(authMiddleware)

	dunning.GET("", handler.List)
	dunning.GET("/metrics", handler.Metrics)
	dunning.GET("/:id", handler.Get)
}
