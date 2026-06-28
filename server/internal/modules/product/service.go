package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidProduct = errors.New("invalid product")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Product, error) {
	return s.repo.Create(ctx, userID, req)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Product, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Product, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Product, error) {
	return s.repo.Update(ctx, userID, id, req)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) UpdateBenefits(ctx context.Context, userID, productID uuid.UUID, req UpdateBenefitsRequest) (*Product, error) {
	if userID == uuid.Nil || productID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid product id", ErrInvalidProduct)
	}
	for _, benefitID := range req.Benefits {
		if benefitID == uuid.Nil {
			return nil, fmt.Errorf("%w: invalid benefit id", ErrInvalidProduct)
		}
	}

	return s.repo.UpdateBenefits(ctx, userID, productID, req.Benefits)
}
