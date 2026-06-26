package pat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const tokenBytes = 32

var (
	ErrInvalidToken  = errors.New("invalid personal access token")
	ErrInvalidExpiry = errors.New("personal access token expiry must be in the future")
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*CreateResponse, error) {
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidExpiry
	}

	rawToken, err := newRawToken()
	if err != nil {
		return nil, fmt.Errorf("create personal access token: %w", err)
	}

	token, err := s.repository.Create(ctx, userID, CreateParams{
		Name:      strings.TrimSpace(req.Name),
		TokenHash: HashToken(rawToken),
		ExpiresAt: req.ExpiresAt,
		Metadata:  req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	return &CreateResponse{Token: *token, RawToken: rawToken}, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Token, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Revoke(ctx context.Context, userID, tokenID uuid.UUID) error {
	return s.repository.Revoke(ctx, userID, tokenID)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Token, error) {
	if !strings.HasPrefix(rawToken, TokenPrefix) {
		return nil, ErrInvalidToken
	}

	return s.repository.GetActiveByHash(ctx, HashToken(rawToken))
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func newRawToken() (string, error) {
	bytes := make([]byte, tokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return TokenPrefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}
