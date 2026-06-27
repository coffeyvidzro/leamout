package transaction

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidTransaction = errors.New("invalid transaction")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*Transaction, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidTransaction
	}
	if params.Type == "" || params.Status == "" || strings.TrimSpace(params.Currency) == "" || params.Amount < 0 {
		return nil, ErrInvalidTransaction
	}
	return s.repository.Create(ctx, params)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Transaction, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrInvalidTransaction
	}
	return s.repository.Get(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, params ListParams) ([]Transaction, error) {
	return s.repository.List(ctx, params)
}
