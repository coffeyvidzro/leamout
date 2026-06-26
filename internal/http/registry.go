package http

import (
	"log/slog"

	"github.com/arcjet/arcjet-go"
	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/cuffeyvidzro/leamout/internal/platform/geoip"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg        *config.Config
	log        *slog.Logger
	pgPool     *pgxpool.Pool
	redis      *redis.Client
	geolocator *geoip.Geolocator
	arcjet     *arcjet.Client
}

func NewServer(
	cfg *config.Config,
	log *slog.Logger,
	pgPool *pgxpool.Pool,
	redis *redis.Client,
	geolocator *geoip.Geolocator,
	arcjetClient *arcjet.Client,
) *Server {
	return &Server{
		cfg:        cfg,
		log:        log,
		pgPool:     pgPool,
		redis:      redis,
		geolocator: geolocator,
		arcjet:     arcjetClient,
	}
}

func (s *Server) oauthRegistry() *oauth.Registry {
	return oauth.NewRegistry(
		oauth.NewGoogle(oauth.ProviderConfig{
			ClientID:     s.cfg.Google.ClientID,
			ClientSecret: s.cfg.Google.ClientSecret,
			RedirectURL:  s.cfg.APIBaseURL + "/v1/auth/google/callback",
		}),
		oauth.NewGitHub(oauth.ProviderConfig{
			ClientID:     s.cfg.Github.ClientID,
			ClientSecret: s.cfg.Github.ClientSecret,
			RedirectURL:  s.cfg.APIBaseURL + "/v1/auth/github/callback",
		}),
	)
}
