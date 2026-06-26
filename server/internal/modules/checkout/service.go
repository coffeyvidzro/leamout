package checkout

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
	"github.com/google/uuid"
)

const clientSecretBytes = 32

type PaymentInitiator interface {
	InitiatePayment(ctx context.Context, req payment.InitiatePaymentRequest) (*payment.InitiatePaymentResult, error)
}

type Service struct {
	repository     *Repository
	paymentService PaymentInitiator
	webhookURLFor  func(provider string) string
}

func NewService(repository *Repository, paymentService PaymentInitiator, webhookURLFor func(provider string) string) *Service {
	return &Service{repository: repository, paymentService: paymentService, webhookURLFor: webhookURLFor}
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

func (s *Service) Pay(ctx context.Context, clientSecret string, req PayRequest) (*PayResponse, error) {
	if s.paymentService == nil {
		return nil, errors.New("payment service is not configured")
	}

	session, err := s.GetPublic(ctx, clientSecret)
	if err != nil {
		return nil, err
	}

	externalRef := uuid.NewString()
	metadata := map[string]string{"checkout_session_id": session.ID.String(), "user_id": session.UserID.String()}
	for key, value := range stringMetadata(session.Metadata) {
		if _, exists := metadata[key]; !exists {
			metadata[key] = value
		}
	}

	result, err := s.paymentService.InitiatePayment(ctx, payment.InitiatePaymentRequest{
		UserID:            session.UserID.String(),
		ExternalRef:       externalRef,
		AmountMinor:       session.Amount,
		Currency:          session.Currency,
		Country:           req.Country,
		Method:            provider.PaymentMethodMobileMoney,
		Operator:          payment.MobileMoneyOperator(req.Operator),
		PreferredProvider: provider.ID(req.PreferredProvider),
		Description:       labelOrDefault(session.Label),
		Customer: payment.Customer{
			Name:    strings.TrimSpace(req.CustomerName),
			Email:   strings.TrimSpace(req.CustomerEmail),
			Phone:   strings.TrimSpace(req.Phone),
			Country: strings.TrimSpace(req.Country),
		},
		CallbackURL: callbackURL(s.webhookURLFor, req.PreferredProvider),
		ReturnURL:   returnURL(session),
		Metadata:    metadata,
	})
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreatePaymentAttempt(ctx, CreatePaymentAttemptParams{
		CheckoutSessionID: session.ID,
		UserID:            session.UserID,
		ExternalRef:       result.ExternalRef,
		ProviderID:        string(result.ProviderID),
		ProviderReference: result.ProviderReference,
		Status:            PaymentAttemptStatus(result.Status),
		Amount:            session.Amount,
		Currency:          session.Currency,
		Country:           strings.ToUpper(strings.TrimSpace(req.Country)),
		PaymentMethod:     string(provider.PaymentMethodMobileMoney),
		Operator:          strings.ToLower(strings.TrimSpace(req.Operator)),
		CustomerPhone:     payment.NormalizePhone(req.Country, req.Phone),
		ProviderResponse:  result.ProviderResponse,
		Metadata:          metadata,
	}); err != nil {
		return nil, err
	}

	return &PayResponse{
		CheckoutSessionID: session.ID.String(),
		ExternalRef:       result.ExternalRef,
		ProviderID:        string(result.ProviderID),
		ProviderReference: result.ProviderReference,
		Status:            string(result.Status),
		NextActionType:    string(result.NextActionType),
		NextActionURL:     result.NextActionURL,
		CustomerMessage:   result.CustomerMessage,
	}, nil
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

func labelOrDefault(label *string) string {
	if label == nil || strings.TrimSpace(*label) == "" {
		return "Leamout payment"
	}
	return strings.TrimSpace(*label)
}

func returnURL(session *Session) string {
	if session == nil {
		return ""
	}
	if session.ReturnURL != nil && strings.TrimSpace(*session.ReturnURL) != "" {
		return strings.TrimSpace(*session.ReturnURL)
	}
	if session.SuccessURL != nil && strings.TrimSpace(*session.SuccessURL) != "" {
		return strings.TrimSpace(*session.SuccessURL)
	}
	return ""
}

func callbackURL(fn func(provider string) string, preferredProvider string) string {
	if fn == nil || strings.TrimSpace(preferredProvider) == "" {
		return ""
	}
	return fn(strings.TrimSpace(preferredProvider))
}

func stringMetadata(metadata map[string]any) map[string]string {
	out := make(map[string]string)
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprint(value))
	}
	return out
}
