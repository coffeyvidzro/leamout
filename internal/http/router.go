package http

import (
	nethttp "net/http"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/gin-gonic/gin"
)

func (s *Server) Router() *gin.Engine {
	if !s.cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())

	router.Use(middleware.RequestContext())
	router.Use(middleware.RequestLogger(s.log))
	router.Use(middleware.Secure(s.cfg.IsDevelopment()))
	router.Use(middleware.CORS(s.cfg.CORSOrigins, s.cfg.IsDevelopment()))

	auth.RegisterRoutes(router, s.authHandler())
	session.RegisterRoutes(router, s.sessionHandler(), middleware.RequireAuth(s.authRepository()))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	return router
}
