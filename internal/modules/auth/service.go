package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
)

var (
	ErrUnverifiedEmail = errors.New("oauth profile email is not verified")
)

type Service struct {
	repository     *Repository
	oauthRegistry  *oauth.Registry
	sessionService *session.Service
}

func NewService(repository *Repository, oauthRegistry *oauth.Registry, sessionService *session.Service) *Service {
	return &Service{
		repository:     repository,
		oauthRegistry:  oauthRegistry,
		sessionService: sessionService,
	}
}

func (s *Service) NewOAuthState() (string, error) {
	return newToken(32)
}

func (s *Service) OAuthURL(providerName, state string) (string, error) {
	provider, err := s.oauthRegistry.Get(providerName)
	if err != nil {
		return "", err
	}

	return provider.AuthURL(state), nil
}

func (s *Service) CompleteOAuthLogin(ctx context.Context, request OAuthLoginRequest) (*AuthResponse, string, error) {
	provider, err := s.oauthRegistry.Get(request.Provider)
	if err != nil {
		return nil, "", err
	}

	profile, err := provider.Exchange(ctx, request.Code)
	if err != nil {
		return nil, "", err
	}
	if !profile.EmailVerified {
		return nil, "", ErrUnverifiedEmail
	}

	user, err := s.repository.UpsertOAuthUser(ctx, profile)
	if err != nil {
		return nil, "", err
	}

	token, sess, err := s.sessionService.CreateSession(ctx, user.ID, request.UserAgent, request.IPAddress)
	if err != nil {
		return nil, "", err
	}

	return &AuthResponse{
		User: AuthUser{
			ID:            user.ID,
			Name:          user.Name,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			AvatarURL:     user.AvatarURL,
			Status:        user.Status,
		},
		Session: AuthSession{
			ID:        sess.ID,
			UserID:    sess.UserID,
			ExpiresAt: sess.ExpiresAt,
			CreatedAt: sess.CreatedAt,
		},
	}, token, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	return s.sessionService.RevokeByToken(ctx, sessionToken)
}

func newToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
