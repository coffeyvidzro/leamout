package customer

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Customer, error) {
	return s.repo.Create(ctx, userID, req)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Customer, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Customer, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) GetState(ctx context.Context, userID, id uuid.UUID) (*State, error) {
	return s.repo.GetState(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Customer, error) {
	return s.repo.Update(ctx, userID, id, req)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) GetByExternalID(ctx context.Context, userID uuid.UUID, externalID string) (*Customer, error) {
	return s.repo.GetByExternalID(ctx, userID, strings.TrimSpace(externalID))
}

func (s *Service) GetStateByExternalID(ctx context.Context, userID uuid.UUID, externalID string) (*State, error) {
	return s.repo.GetStateByExternalID(ctx, userID, strings.TrimSpace(externalID))
}

func (s *Service) UpdateByExternalID(ctx context.Context, userID uuid.UUID, externalID string, req UpdateRequest) (*Customer, error) {
	return s.repo.UpdateByExternalID(ctx, userID, strings.TrimSpace(externalID), req)
}

func (s *Service) DeleteByExternalID(ctx context.Context, userID uuid.UUID, externalID string) error {
	return s.repo.DeleteByExternalID(ctx, userID, strings.TrimSpace(externalID))
}
