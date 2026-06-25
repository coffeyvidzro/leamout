package http

import (
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	pgPool *pgxpool.Pool
	redis  *redis.Client
}

func NewServer(cfg *config.Config, log *slog.Logger, pgPool *pgxpool.Pool, redis *redis.Client) *Server {
	return &Server{
		cfg:    cfg,
		log:    log,
		pgPool: pgPool,
		redis:  redis,
	}
}

func (s *Server) oauthRegistry() *oauth.Registry {
	return oauth.NewRegistry(
		oauth.NewGoogle(oauth.ProviderConfig{
			ClientID:     s.cfg.Google.ClientID,
			ClientSecret: s.cfg.Google.ClientSecret,
			RedirectURL:  s.cfg.BaseURL + "/v1/auth/google/callback",
		}),
		oauth.NewGitHub(oauth.ProviderConfig{
			ClientID:     s.cfg.Github.ClientID,
			ClientSecret: s.cfg.Github.ClientSecret,
			RedirectURL:  s.cfg.BaseURL + "/v1/auth/github/callback",
		}),
	)
}
