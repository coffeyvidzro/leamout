package http

import (
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/modules/customer"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/product"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/modules/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	postgres *pgxpool.Pool
	redis    *redis.Client
}

func NewServer(cfg *config.Config, log *slog.Logger, postgres *pgxpool.Pool, redis *redis.Client) *Server {
	return &Server{
		cfg:      cfg,
		log:      log,
		postgres: postgres,
		redis:    redis,
	}
}

func (s *Server) authRepository() *auth.Repository {
	return auth.NewRepository(s.postgres)
}

func (s *Server) authHandler() *auth.Handler {
	repository := s.authRepository()
	service := auth.NewService(repository, s.oauthRegistry(), s.sessionService())

	return auth.NewHandler(service, s.cfg.IsDevelopment())
}

func (s *Server) sessionHandler() *session.Handler {
	return session.NewHandler(s.sessionService())
}

func (s *Server) sessionService() *session.Service {
	repository := session.NewRepository(s.postgres, s.redis)
	return session.NewService(repository)
}

func (s *Server) userHandler() *user.Handler {
	repository := user.NewRepository(s.postgres)
	service := user.NewService(repository)

	return user.NewHandler(service)
}

func (s *Server) customerHandler() *customer.Handler {
	repository := customer.NewRepository(s.postgres)
	service := customer.NewService(repository)

	return customer.NewHandler(service)
}

func (s *Server) productHandler() *product.Handler {
	repository := product.NewRepository(s.postgres)
	service := product.NewService(repository)

	return product.NewHandler(service)
}

func (s *Server) subscriptionHandler() *subscription.Handler {
	repository := subscription.NewRepository(s.postgres)
	service := subscription.NewService(repository)

	return subscription.NewHandler(service)
}

func (s *Server) checkoutHandler() *checkout.Handler {
	return checkout.NewHandler(s.checkoutService())
}

func (s *Server) creditsHandler() *credits.Handler {
	repository := credits.NewRepository(s.postgres)
	service := credits.NewService(repository)

	return credits.NewHandler(service)
}

func (s *Server) checkoutService() *checkout.Service {
	repository := checkout.NewRepository(s.postgres)

	return checkout.NewService(repository)
}

func (s *Server) dunningHandler() *dunning.Handler {
	repository := dunning.NewRepository(s.postgres)
	service := dunning.NewService(repository, s.checkoutService())

	return dunning.NewHandler(service)
}

func (s *Server) oauthRegistry() *oauth.Registry {
	return oauth.NewRegistry(
		oauth.NewGoogle(oauth.ProviderConfig{
			ClientID:     s.cfg.Google.ClientID,
			ClientSecret: s.cfg.Google.ClientSecret,
			RedirectURL:  s.cfg.BaseURL + "/auth/google/callback",
		}),
		oauth.NewGitHub(oauth.ProviderConfig{
			ClientID:     s.cfg.Github.ClientID,
			ClientSecret: s.cfg.Github.ClientSecret,
			RedirectURL:  s.cfg.BaseURL + "/auth/github/callback",
		}),
	)
}
