package checkout

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	corepayment "github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/google/uuid"
)

const clientSecretBytes = 32

var (
	ErrInvalidCheckoutRequest = errors.New("invalid checkout request")
	ErrInvalidPaymentRequest  = errors.New("invalid checkout payment request")
)

type PaymentCharger interface {
	Charge(ctx context.Context, payload corepayment.UnifiedPayload) (*corepayment.ChargeResult, error)
}

type Service struct {
	repository     *Repository
	paymentService PaymentCharger
}

func NewService(repository *Repository, paymentService PaymentCharger) *Service {
	return &Service{
		repository:     repository,
		paymentService: paymentService,
	}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Session, error) {
	if s.repository == nil {
		return nil, errors.New("checkout repository is not configured")
	}

	if req.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("%w: expires_at is required", ErrInvalidCheckoutRequest)
	}
	if !req.ExpiresAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidCheckoutRequest)
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidCheckoutRequest)
	}
	req.Currency = normalizeCurrency(req.Currency)
	if len(req.Currency) != 3 {
		return nil, fmt.Errorf("%w: currency must be a 3-letter ISO code", ErrInvalidCheckoutRequest)
	}

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
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return nil, ErrNotFound
	}

	return s.repository.GetByClientSecretHash(ctx, HashClientSecret(clientSecret))
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Session, error) {
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidCheckoutRequest)
	}

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

	country := normalizeCountry(req.Country)
	if country == "" {
		return nil, fmt.Errorf("%w: unsupported country %q", ErrInvalidPaymentRequest, req.Country)
	}

	network := normalizeNetwork(req.Network)
	if network == "" {
		return nil, fmt.Errorf("%w: network is required", ErrInvalidPaymentRequest)
	}

	currency := normalizeCurrency(session.Currency)
	phone := normalizePhone(country, req.Phone)
	if phone == "" {
		return nil, fmt.Errorf("%w: phone is required", ErrInvalidPaymentRequest)
	}

	transactionID := uuid.NewString()
	metadata := stringMetadata(session.Metadata)
	addPaymentMetadata(metadata, session, req, country, network, phone)

	result, err := s.paymentService.Charge(ctx, corepayment.UnifiedPayload{
		TransactionID: transactionID,
		Country:       country,
		Network:       network,
		PhoneNumber:   phone,
		Amount:        strconv.FormatInt(session.Amount, 10),
		Currency:      currency,
		Metadata:      metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPaymentRequest, err)
	}
	if result == nil {
		return nil, errors.New("payment service returned empty result")
	}

	return &PayResponse{
		CheckoutSessionID: session.ID.String(),
		TransactionID:     firstNonEmpty(result.TransactionID, transactionID),
		Provider:          string(result.Provider),
		ProviderReference: result.ProviderReference,
		Status:            string(result.Status),
		Amount:            session.Amount,
		Currency:          currency,
		Country:           country,
		Network:           network,
		Phone:             phone,
		CustomerMessage:   result.Message,
	}, nil
}

func (s *Service) CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error {
	return s.repository.CompletePaidCheckout(ctx, checkoutID)
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

func addPaymentMetadata(metadata map[string]string, session *Session, req PayRequest, country, network, phone string) {
	if metadata == nil || session == nil {
		return
	}

	metadata["checkout_session_id"] = session.ID.String()
	metadata["checkout_user_id"] = session.UserID.String()
	metadata["checkout_mode"] = string(session.Mode)
	metadata["checkout_source"] = string(session.Source)
	metadata["checkout_amount"] = strconv.FormatInt(session.Amount, 10)
	metadata["checkout_currency"] = normalizeCurrency(session.Currency)
	metadata["checkout_country"] = country
	metadata["checkout_network"] = network
	metadata["customer_phone"] = phone

	if session.CustomerID != nil {
		metadata["checkout_customer_id"] = session.CustomerID.String()
	}
	if session.SubscriptionID != nil {
		metadata["checkout_subscription_id"] = session.SubscriptionID.String()
	}
	if session.Label != nil && strings.TrimSpace(*session.Label) != "" {
		metadata["checkout_label"] = strings.TrimSpace(*session.Label)
	}
	if name := strings.TrimSpace(req.CustomerName); name != "" {
		metadata["customer_name"] = name
	}
	if email := strings.TrimSpace(req.CustomerEmail); email != "" {
		metadata["customer_email"] = email
	}
	if detectedCountry := normalizeCountry(req.Intelligence.DetectedCountry); detectedCountry != "" {
		metadata["detected_country"] = detectedCountry
	}
	if source := strings.TrimSpace(req.Intelligence.DetectedSource); source != "" {
		metadata["detected_country_source"] = source
	}
	if ip := strings.TrimSpace(req.Intelligence.ClientIP); ip != "" {
		metadata["client_ip"] = ip
	}
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeCountry(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.Join(strings.Fields(value), " ")

	switch value {
	case "BEN", "BJ", "BENIN":
		return "BEN"
	case "BFA", "BF", "BURKINA FASO":
		return "BFA"
	case "CMR", "CM", "CAMEROON":
		return "CMR"
	case "GHA", "GH", "GHANA":
		return "GHA"
	case "CIV", "CI", "IVORY COAST", "COTE DIVOIRE", "CÔTE DIVOIRE", "COTE D'IVOIRE", "CÔTE D'IVOIRE":
		return "CIV"
	case "NGA", "NG", "NIGERIA":
		return "NGA"
	case "SEN", "SN", "SENEGAL":
		return "SEN"
	case "SLE", "SL", "SIERRA LEONE":
		return "SLE"
	default:
		return ""
	}
}

func normalizeNetwork(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")

	switch value {
	case "MTN", "MTNMOMO", "MOMO":
		return "MTN"
	case "MOOV":
		return "MOOV"
	case "ORANGE":
		return "ORANGE"
	case "FREE":
		return "FREE"
	case "TELECEL", "VODAFONE":
		return "TELECEL"
	case "AIRTELTIGO", "AIRTELTIGOMONEY", "AT", "ATMONEY":
		return "AIRTELTIGO"
	default:
		return value
	}
}

func normalizePhone(country, phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.TrimPrefix(phone, "+")

	if strings.HasPrefix(phone, "00") && len(phone) > 2 {
		phone = phone[2:]
	}

	prefix := phonePrefixForCountry(country)
	if prefix == "" || phone == "" {
		return phone
	}

	if strings.HasPrefix(phone, prefix) {
		return phone
	}

	if strings.HasPrefix(phone, "0") && len(phone) > 1 {
		return prefix + phone[1:]
	}

	return prefix + phone
}

func phonePrefixForCountry(country string) string {
	switch normalizeCountry(country) {
	case "BEN":
		return "229"
	case "BFA":
		return "226"
	case "CMR":
		return "237"
	case "GHA":
		return "233"
	case "CIV":
		return "225"
	case "NGA":
		return "234"
	case "SEN":
		return "221"
	case "SLE":
		return "232"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
