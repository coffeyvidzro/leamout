package dunning

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	// Public system-generated recovery link. Example: https://leam.out/r/<token>.
	router.GET("/r/:token", handler.OpenRecoveryLink)

	// Protected merchant visibility. Dunning attempts are system-created by jobs.
	dunning := router.Group("/dunning")
	dunning.Use(authMiddleware)

	dunning.GET("", handler.List)
	dunning.GET("/:id", handler.Get)
}
