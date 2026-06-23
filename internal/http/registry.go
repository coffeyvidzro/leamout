package http

import (
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
)

type Server struct {
	cfg *config.Config
	log *slog.Logger
}

func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	return &Server{
		cfg: cfg,
		log: log,
	}
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
