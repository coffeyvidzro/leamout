package checkout

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler) {
	router.GET("/r/:token", handler.Show)
	router.POST("/r/:token/pay", handler.MockPay)
}
