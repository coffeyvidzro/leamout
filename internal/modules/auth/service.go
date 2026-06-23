package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
)

const (
	oauthStateTTL     = 10 * time.Minute
	sessionTTL        = 30 * 24 * time.Hour
	sessionTokenBytes = 32
)

var (
	ErrUnverifiedEmail = errors.New("oauth profile email is not verified")
	ErrUserNotFound    = errors.New("user not found")
)

type Service struct {
	oauthRegistry *oauth.Registry
	repository    Repository
	stateStore    StateStore
}

func NewService(oauthRegistry *oauth.Registry, repository Repository, stateStore StateStore) *Service {
	return &Service{
		oauthRegistry: oauthRegistry,
		repository:    repository,
		stateStore:    stateStore,
	}
}

func (s *Service) Login(ctx context.Context, providerName string) (string, error) {
	provider, err := s.oauthRegistry.Get(providerName)
	if err != nil {
		return "", err
	}

	state, err := newToken(32)
	if err != nil {
		return "", fmt.Errorf("create oauth state: %w", err)
	}

	if err := s.stateStore.SaveOAuthState(ctx, providerName, state, oauthStateTTL); err != nil {
		return "", err
	}

	return provider.AuthURL(state), nil
}

func (s *Service) CompleteOAuthLogin(ctx context.Context, request OAuthLoginRequest) (*AuthResponse, string, error) {
	provider, err := s.oauthRegistry.Get(request.Provider)
	if err != nil {
		return nil, "", err
	}

	if err := s.stateStore.ConsumeOAuthState(ctx, request.Provider, request.State); err != nil {
		return nil, "", err
	}

	profile, err := provider.Exchange(ctx, request.Code)
	if err != nil {
		return nil, "", err
	}
	if !profile.EmailVerified {
		return nil, "", ErrUnverifiedEmail
	}

	user, err := s.userForProfile(ctx, profile)
	if err != nil {
		return nil, "", err
	}

	rawToken, err := newToken(sessionTokenBytes)
	if err != nil {
		return nil, "", fmt.Errorf("create session token: %w", err)
	}

	session, err := s.repository.CreateSession(ctx, CreateSessionParams{
		UserID:    user.ID,
		TokenHash: HashSessionToken(rawToken),
		UserAgent: request.UserAgent,
		IPAddress: request.IPAddress,
		ExpiresAt: time.Now().Add(sessionTTL),
	})
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
			ID:        session.ID,
			UserID:    session.UserID,
			ExpiresAt: session.ExpiresAt,
			CreatedAt: session.CreatedAt,
		},
	}, rawToken, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	if err := s.repository.RevokeSessionByTokenHash(ctx, HashSessionToken(rawToken)); err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}

	return nil
}

func (s *Service) userForProfile(ctx context.Context, profile *oauth.Profile) (*User, error) {
	account, err := s.repository.FindAccount(ctx, profile.Provider, profile.ProviderUserID)
	if err != nil {
		return nil, err
	}
	if account != nil {
		user, err := s.repository.FindUserByID(ctx, account.UserID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrUserNotFound
		}
		return user, nil
	}

	user, err := s.repository.FindUserByEmail(ctx, profile.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user, err = s.repository.CreateUser(ctx, profile)
		if err != nil {
			return nil, err
		}
	}

	if _, err := s.repository.CreateAccount(ctx, user.ID, profile); err != nil {
		return nil, err
	}

	return user, nil
}

func HashSessionToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func newToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
