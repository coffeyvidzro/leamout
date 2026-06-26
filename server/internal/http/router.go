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

func (s *Server) BuildEngine() (*gin.Engine, error) {
	if !s.cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	if err := router.SetTrustedProxies(s.cfg.TrustedProxies); err != nil {
		return nil, err
	}

	// Global Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestContext())
	router.Use(middleware.RequestLogger(s.log))
	router.Use(middleware.Arcjet(s.arcjet, s.log))
	router.Use(middleware.Secure(s.cfg.IsDevelopment()))
	router.Use(middleware.CORS(s.cfg.CORSOrigins, s.cfg.IsDevelopment()))
	router.Use(middleware.Geolocation(s.geolocator, s.log))

	// Initialize Repositories

	userRepo := user.NewRepository(s.pgPool)
	sessionRepo := session.NewRepository(s.pgPool, s.redis)
	authRepo := auth.NewRepository(s.pgPool)
	customerRepo := customer.NewRepository(s.pgPool)
	productRepo := product.NewRepository(s.pgPool)
	checkoutRepo := checkout.NewRepository(s.pgPool)
	patRepo := pat.NewRepository(s.pgPool)
	subscriptionRepo := subscription.NewRepository(s.pgPool)
	creditsRepo := credits.NewRepository(s.pgPool)
	dunningRepo := dunning.NewRepository(s.pgPool)

	//  Initialize Services
	userService := user.NewService(userRepo)
	sessionService := session.NewService(sessionRepo)
	authService := auth.NewService(authRepo, s.oauthRegistry(), sessionService)
	customerService := customer.NewService(customerRepo)
	productService := product.NewService(productRepo)
	checkoutService := checkout.NewService(checkoutRepo)
	patService := pat.NewService(patRepo)
	subscriptionService := subscription.NewService(subscriptionRepo)
	creditsService := credits.NewService(creditsRepo)
	dunningService := dunning.NewService(dunningRepo, checkoutService)

	// Initialize Handlers
	userHandler := user.NewHandler(userService)
	sessionHandler := session.NewHandler(sessionService)
	authHandler := auth.NewHandler(authService, s.cfg.IsDevelopment())
	customerHandler := customer.NewHandler(customerService)
	productHandler := product.NewHandler(productService)
	checkoutHandler := checkout.NewHandler(checkoutService)
	patHandler := pat.NewHandler(patService)
	subscriptionHandler := subscription.NewHandler(subscriptionService)
	creditsHandler := credits.NewHandler(creditsService)
	dunningHandler := dunning.NewHandler(dunningService, s.cfg.FrontendBaseURL)

	//  Initialize Middleware
	sessionAuthMiddleware := middleware.SessionAuthMiddleware(sessionService)
	authMiddleware := middleware.AuthMiddleware(sessionService, patService)

	// Register Routes
	v1 := router.Group("/v1")
	{
		auth.RegisterRoutes(v1, authHandler, sessionAuthMiddleware)
		pat.RegisterRoutes(v1, patHandler, sessionAuthMiddleware)
		session.RegisterRoutes(v1, sessionHandler, authMiddleware)
		user.RegisterRoutes(v1, userHandler, authMiddleware)
		customer.RegisterRoutes(v1, customerHandler, authMiddleware)
		product.RegisterRoutes(v1, productHandler, authMiddleware)
		subscription.RegisterRoutes(v1, subscriptionHandler, authMiddleware)
		checkout.RegisterRoutes(v1, checkoutHandler, authMiddleware)
		credits.RegisterRoutes(v1, creditsHandler, authMiddleware)
		dunning.RegisterRoutes(v1, dunningHandler, authMiddleware)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	return router, nil
}
