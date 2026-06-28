package http

import (
	nethttp "net/http"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/benefit"
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/modules/customer"
	"github.com/cuffeyvidzro/leamout/internal/modules/customermeter"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/meter"
	"github.com/cuffeyvidzro/leamout/internal/modules/pat"
	modulepayment "github.com/cuffeyvidzro/leamout/internal/modules/payment"
	"github.com/cuffeyvidzro/leamout/internal/modules/product"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/cuffeyvidzro/leamout/internal/modules/usageevent"
	"github.com/cuffeyvidzro/leamout/internal/modules/user"
	"github.com/cuffeyvidzro/leamout/internal/modules/wallet"
	paymentwebhook "github.com/cuffeyvidzro/leamout/internal/payment/webhook"
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

	router.Use(gin.Recovery())
	router.Use(middleware.RequestContext())
	router.Use(middleware.RequestLogger(s.log))
	router.Use(middleware.Arcjet(s.arcjet, s.log))
	router.Use(middleware.Secure(s.cfg.IsDevelopment()))
	router.Use(middleware.CORS(s.cfg.CORSOrigins, s.cfg.IsDevelopment()))
	router.Use(middleware.Geolocation(s.geolocator, s.log))

	userRepo := user.NewRepository(s.pgPool)
	sessionRepo := session.NewRepository(s.pgPool, s.redis)
	authRepo := auth.NewRepository(s.pgPool)
	customerRepo := customer.NewRepository(s.pgPool)
	customerMeterRepo := customermeter.NewRepository(s.pgPool)
	productRepo := product.NewRepository(s.pgPool)
	checkoutRepo := checkout.NewRepository(s.pgPool, customerMeterRepo)
	patRepo := pat.NewRepository(s.pgPool)
	subscriptionRepo := subscription.NewRepository(s.pgPool)
	creditsRepo := credits.NewRepository(s.pgPool)
	dunningRepo := dunning.NewRepository(s.pgPool)
	transactionRepo := transaction.NewRepository(s.pgPool)
	walletRepo := wallet.NewRepository(s.pgPool)
	paymentRepo := modulepayment.NewRepository(s.pgPool)
	usageRepo := usageevent.NewRepository(s.pgPool)
	meterRepo := meter.NewRepository(s.pgPool)
	benefitRepo := benefit.NewRepository(s.pgPool)

	transactionService := transaction.NewService(transactionRepo)
	walletService := wallet.NewService(walletRepo)

	paymentStack := s.buildPaymentStack(
		checkoutRepo,
		paymentRepo,
		transactionService,
		walletService,
	)

	userService := user.NewService(userRepo)
	sessionService := session.NewService(sessionRepo)
	authService := auth.NewService(authRepo, s.oauthRegistry(), sessionService)
	customerService := customer.NewService(customerRepo)
	customerMeterService := customermeter.NewService(customerMeterRepo)
	productService := product.NewService(productRepo)
	patService := pat.NewService(patRepo)
	subscriptionService := subscription.NewService(subscriptionRepo)
	creditsService := credits.NewService(creditsRepo)
	dunningService := dunning.NewService(dunningRepo, paymentStack.CheckoutService)
	usageService := usageevent.NewService(usageRepo)
	meterService := meter.NewService(meterRepo)
	benefitService := benefit.NewService(benefitRepo)

	userHandler := user.NewHandler(userService)
	sessionHandler := session.NewHandler(sessionService)
	authHandler := auth.NewHandler(authService, s.cfg.IsDevelopment())
	customerHandler := customer.NewHandler(customerService)
	customerMeterHandler := customermeter.NewHandler(customerMeterService)
	productHandler := product.NewHandler(productService)
	checkoutHandler := paymentStack.CheckoutHandler
	patHandler := pat.NewHandler(patService)
	subscriptionHandler := subscription.NewHandler(subscriptionService)
	creditsHandler := credits.NewHandler(creditsService)
	dunningHandler := dunning.NewHandler(dunningService, s.cfg.FrontendBaseURL)
	transactionHandler := transaction.NewHandler(transactionService)
	walletHandler := wallet.NewHandler(walletService)
	paymentHandler := paymentStack.PaymentHandler
	usageHandler := usageevent.NewHandler(usageService)
	meterHandler := meter.NewHandler(meterService)
	benefitHandler := benefit.NewHandler(benefitService)

	paymentWebhookRegistry, err := s.paymentWebhookRegistry()
	if err != nil {
		return nil, err
	}

	paymentWebhookHandler := paymentwebhook.NewHandler(paymentWebhookRegistry)

	sessionAuthMiddleware := middleware.SessionAuthMiddleware(sessionService)
	authMiddleware := middleware.AuthMiddleware(sessionService, patService)

	v1 := router.Group("/v1")
	{
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/:provider", paymentWebhookHandler.Handle)
			webhooks.POST("/:provider/:eventType", paymentWebhookHandler.Handle)
		}

		auth.RegisterRoutes(v1, authHandler, sessionAuthMiddleware)
		pat.RegisterRoutes(v1, patHandler, sessionAuthMiddleware)
		session.RegisterRoutes(v1, sessionHandler, authMiddleware)
		user.RegisterRoutes(v1, userHandler, authMiddleware)
		customer.RegisterRoutes(v1, customerHandler, authMiddleware)
		customermeter.RegisterRoutes(v1, customerMeterHandler, authMiddleware)
		product.RegisterRoutes(v1, productHandler, authMiddleware)
		subscription.RegisterRoutes(v1, subscriptionHandler, authMiddleware)
		checkout.RegisterRoutes(v1, checkoutHandler, authMiddleware)
		credits.RegisterRoutes(v1, creditsHandler, authMiddleware)
		dunning.RegisterRoutes(v1, dunningHandler, authMiddleware)
		transaction.RegisterRoutes(v1, transactionHandler, authMiddleware)
		wallet.RegisterRoutes(v1, walletHandler, authMiddleware)
		modulepayment.RegisterRoutes(v1, paymentHandler, authMiddleware)
		usageevent.RegisterRoutes(v1, usageHandler, authMiddleware)
		meter.RegisterRoutes(v1, meterHandler, authMiddleware)
		benefit.RegisterRoutes(v1, benefitHandler, authMiddleware)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	return router, nil
}
