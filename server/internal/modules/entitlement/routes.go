package entitlement

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	entitlements := router.Group("/entitlements")
	entitlements.Use(authMiddleware)

	entitlements.POST("/check", handler.Check)
}
