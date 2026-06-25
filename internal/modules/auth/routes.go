package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler, authMiddleware gin.HandlerFunc) {
	auth := router.Group("/auth")
	{
		auth.GET("/google", handler.Google)
		auth.GET("/google/callback", handler.GoogleCallback)

		auth.GET("/github", handler.GitHub)
		auth.GET("/github/callback", handler.GitHubCallback)
	}

	protected := router.Group("/auth")
	protected.Use(authMiddleware)
	{
		protected.POST("/logout", handler.Logout)
	}
}
