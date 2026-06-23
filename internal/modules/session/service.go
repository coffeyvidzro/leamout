package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	sessionTTL        = 30 * 24 * time.Hour
	sessionTokenBytes = 32
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (string, error) {
	rawToken, err := newToken(sessionTokenBytes)
	if err != nil {
		return "", fmt.Errorf("create session token: %w", err)
	}

	if _, err := s.repository.Create(ctx, CreateParams{
		UserID:    userID,
		RawToken:  rawToken,
		TokenHash: HashToken(rawToken),
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: time.Now().Add(sessionTTL),
	}); err != nil {
		return "", err
	}

	return rawToken, nil
}

func (s *Service) GetByToken(ctx context.Context, rawToken string) (*Session, error) {
	return s.repository.GetByToken(ctx, rawToken)
}

func (s *Service) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.repository.ListByUserID(ctx, userID)
}

func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	return s.repository.GetByID(ctx, sessionID)
}

func (s *Service) RevokeSpecificSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	return s.repository.Delete(ctx, userID, sessionID)
}

func (s *Service) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	return s.repository.DeleteAllByUserID(ctx, userID)
}

func (s *Service) RevokeByToken(ctx context.Context, rawToken string) error {
	return s.repository.DeleteByToken(ctx, rawToken)
}

func HashToken(rawToken string) string {
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
