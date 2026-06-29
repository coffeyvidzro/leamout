package entitlement

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	access := router.Group("/access")
	access.Use(authMiddleware)
	access.POST("/check", handler.Check)

	entitlements := router.Group("/entitlements")
	entitlements.Use(authMiddleware)
	entitlements.POST("/check", handler.Check)
}
