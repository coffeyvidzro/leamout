package http

import (
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	postgres *pgxpool.Pool
}

func NewServer(cfg *config.Config, log *slog.Logger, postgres *pgxpool.Pool) *Server {
	return &Server{
		cfg:      cfg,
		log:      log,
		postgres: postgres,
	}
}

func (s *Server) authRepository() *auth.PostgresRepository {
	return auth.NewPostgresRepository(s.postgres)
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
	repository := session.NewPostgresRepository(s.postgres)
	return session.NewService(repository)
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
