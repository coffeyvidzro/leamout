package http

import (
	nethttp "net/http"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/modules/customer"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/pat"
	"github.com/cuffeyvidzro/leamout/internal/modules/product"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/modules/user"
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
	sessionAuthMiddleware := middleware.SessionAuthMiddleware(s.sessionService())
	authMiddleware := middleware.AuthMiddleware(s.sessionService(), s.patService())
	pat.RegisterRoutes(router, s.patHandler(), sessionAuthMiddleware)
	session.RegisterRoutes(router, s.sessionHandler(), authMiddleware)
	user.RegisterRoutes(router, s.userHandler(), authMiddleware)
	customer.RegisterRoutes(router, s.customerHandler(), authMiddleware)
	product.RegisterRoutes(router, s.productHandler(), authMiddleware)
	subscription.RegisterRoutes(router, s.subscriptionHandler(), authMiddleware)
	checkout.RegisterRoutes(router, s.checkoutHandler(), authMiddleware)
	credits.RegisterRoutes(router, s.creditsHandler(), authMiddleware)
	dunning.RegisterRoutes(router, s.dunningHandler(), authMiddleware)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	return router
}
