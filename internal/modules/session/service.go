package session

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.repository.ListByUserID(ctx, userID)
}

func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	return s.repository.GetByID(ctx, sessionID)
}

func (s *Service) RevokeSpecificSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	return s.repository.RevokeByID(ctx, userID, sessionID)
}

func (s *Service) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	return s.repository.RevokeAllByUserID(ctx, userID)
}
