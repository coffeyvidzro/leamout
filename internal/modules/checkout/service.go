package checkout

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

const clientSecretBytes = 32

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Session, error) {
	clientSecret, err := newClientSecret()
	if err != nil {
		return nil, fmt.Errorf("create checkout client secret: %w", err)
	}

	session, err := s.repository.Create(ctx, userID, req, HashClientSecret(clientSecret))
	if err != nil {
		return nil, err
	}
	session.ClientSecret = clientSecret

	return session, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Session, error) {
	return s.repository.Get(ctx, userID, id)
}

func (s *Service) GetPublic(ctx context.Context, clientSecret string) (*Session, error) {
	return s.repository.GetByClientSecretHash(ctx, HashClientSecret(clientSecret))
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Session, error) {
	return s.repository.Update(ctx, userID, id, req)
}

func (s *Service) Confirm(ctx context.Context, clientSecret string) (*Session, error) {
	return s.repository.ConfirmByClientSecretHash(ctx, HashClientSecret(clientSecret))
}

func HashClientSecret(clientSecret string) string {
	sum := sha256.Sum256([]byte(clientSecret))
	return hex.EncodeToString(sum[:])
}

func newClientSecret() (string, error) {
	bytes := make([]byte, clientSecretBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
