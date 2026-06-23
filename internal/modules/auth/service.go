package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
)

type Service struct {
	oauthRegistry *oauth.Registry
}

func NewService(oauthRegistry *oauth.Registry) *Service {
	return &Service{
		oauthRegistry: oauthRegistry,
	}
}

func (s *Service) Login(ctx context.Context, providerName string) (string, error) {
	provider, err := s.oauthRegistry.Get(providerName)
	if err != nil {
		return "", err
	}

	state, err := newStateToken()
	if err != nil {
		return "", fmt.Errorf("create oauth state: %w", err)
	}

	_ = ctx

	return provider.AuthURL(state), nil
}

func newStateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
