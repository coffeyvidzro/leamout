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
	checkoutsm "github.com/cuffeyvidzro/leamout/internal/platform/statemachine/checkout"
	"github.com/cuffeyvidzro/leamout/pkg/markets"
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

type PaymentRouteResolver interface {
	Resolve(ctx context.Context, payload corepayment.UnifiedPayload) (*corepayment.RoutingResult, error)
}

type Service struct {
	repository     *Repository
	paymentService PaymentCharger
	feeResolver    PaymentRouteResolver
}

func NewService(repository *Repository, paymentService PaymentCharger, feeResolvers ...PaymentRouteResolver) *Service {
	service := &Service{
		repository:     repository,
		paymentService: paymentService,
	}

	if len(feeResolvers) > 0 {
		service.feeResolver = feeResolvers[0]
	} else if resolver, ok := paymentService.(PaymentRouteResolver); ok {
		service.feeResolver = resolver
	}

	return service
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

func (s *Service) GetPublic(ctx context.Context, clientSecret string, req PublicCheckoutRequest) (*PublicCheckoutResponse, error) {
	session, err := s.getPublicSession(ctx, clientSecret)
	if err != nil {
		return nil, err
	}

	return s.publicCheckoutResponse(ctx, session, req)
}

func (s *Service) getPublicSession(ctx context.Context, clientSecret string) (*Session, error) {
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return nil, ErrNotFound
	}

	return s.repository.GetByClientSecretHash(ctx, HashClientSecret(clientSecret))
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Session, error) {
	if req.Status != nil {
		session, err := s.repository.Get(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		if err := checkoutsm.ValidateTransition(checkoutsm.Status(session.Status), checkoutsm.Status(*req.Status)); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCheckoutRequest, err)
		}
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidCheckoutRequest)
	}

	return s.repository.Update(ctx, userID, id, req)
}

func (s *Service) Pay(ctx context.Context, clientSecret string, req PayRequest) (*PayResponse, error) {
	if s.paymentService == nil {
		return nil, errors.New("payment service is not configured")
	}

	session, err := s.getPublicSession(ctx, clientSecret)
	if err != nil {
		return nil, err
	}

	country := markets.NormalizeCountry(req.Country)
	network := markets.NormalizeNetwork(country, req.Network)
	phone := markets.NormalizePhone(country, req.Phone)

	currency := strings.ToUpper(strings.TrimSpace(session.Currency))

	routingFees, err := s.resolveRouteFees(ctx, session, country, network)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPaymentRequest, err)
	}
	fee := calculateCustomerCheckoutFees(session.Amount, routingFees)

	transactionID := uuid.NewString()

	metadata := map[string]string{
		"checkout_user_id":    session.UserID.String(),
		"checkout_session_id": session.ID.String(),
		"checkout_fee_amount": strconv.FormatInt(fee.ProcessingFee, 10),
	}

	if session.CustomerID != nil {
		metadata["checkout_customer_id"] = session.CustomerID.String()
	}

	result, err := s.paymentService.Charge(ctx, corepayment.UnifiedPayload{
		TransactionID: transactionID,
		Country:       country,
		Network:       network,
		PhoneNumber:   phone,
		Amount:        strconv.FormatInt(fee.PayableAmount, 10),
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
		Amount:            fee.PayableAmount,
		BaseAmount:        fee.BaseAmount,
		ProcessingFee:     fee.ProcessingFee,
		PayableAmount:     fee.PayableAmount,
		Fee:               fee,
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

func (s *Service) publicCheckoutResponse(ctx context.Context, session *Session, req PublicCheckoutRequest) (*PublicCheckoutResponse, error) {
	response := &PublicCheckoutResponse{
		ID:         session.ID,
		Mode:       session.Mode,
		Source:     session.Source,
		Label:      session.Label,
		Amount:     session.Amount,
		Currency:   normalizeCurrency(session.Currency),
		Status:     session.Status,
		ExpiresAt:  session.ExpiresAt,
		SuccessURL: session.SuccessURL,
		ReturnURL:  session.ReturnURL,
	}
	hasCountry := strings.TrimSpace(req.Country) != ""
	hasNetwork := strings.TrimSpace(req.Network) != ""
	if !hasCountry && !hasNetwork {
		return response, nil
	}
	if !hasCountry || !hasNetwork {
		return nil, fmt.Errorf("%w: country and network are required to calculate checkout fees", ErrInvalidPaymentRequest)
	}

	country := markets.NormalizeCountry(req.Country)

	network := markets.NormalizeNetwork(country, req.Network)

	routingFees, err := s.resolveRouteFees(ctx, session, country, network)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPaymentRequest, err)
	}

	fee := calculateCustomerCheckoutFees(session.Amount, routingFees)
	response.Country = stringPointer(country)
	response.Network = stringPointer(network)
	response.Fee = &fee

	return response, nil
}

func (s *Service) resolveRouteFees(ctx context.Context, session *Session, country, network string) (corepayment.RoutingFees, error) {
	if s.feeResolver == nil {
		return corepayment.RoutingFees{}, errors.New("payment fee resolver is not configured")
	}

	currency := normalizeCurrency(session.Currency)
	route, err := s.feeResolver.Resolve(ctx, corepayment.UnifiedPayload{
		TransactionID: "checkout_fee_preview",
		Country:       country,
		Network:       network,
		Currency:      currency,
		Amount:        strconv.FormatInt(session.Amount, 10),
	})
	if err != nil {
		return corepayment.RoutingFees{}, err
	}
	if route == nil {
		return corepayment.RoutingFees{}, errors.New("payment route was not resolved")
	}
	if route.Fees.TotalFeeBps <= 0 {
		return corepayment.RoutingFees{}, errors.New("payment route fees are not configured")
	}

	return route.Fees, nil
}

func calculateCustomerCheckoutFees(baseAmount int64, fees corepayment.RoutingFees) CheckoutFeeBreakdown {
	totalFeeBps := fees.TotalFeeBps
	processingFee := percentageCeil(baseAmount, totalFeeBps)
	payableAmount := baseAmount + processingFee

	return CheckoutFeeBreakdown{
		FeePayer:       FeePayerCustomer,
		MMOFeeBps:      fees.MMOFeeBps,
		ProviderFeeBps: fees.ProviderFeeBps,
		TotalFeeBps:    totalFeeBps,
		BaseAmount:     baseAmount,
		ProcessingFee:  processingFee,
		PayableAmount:  payableAmount,
		NetAmount:      baseAmount,
	}
}

func percentageCeil(amount int64, bps int64) int64 {
	if amount <= 0 || bps <= 0 {
		return 0
	}

	return (amount*bps + 9999) / 10000
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

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
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

func stringPointer(value string) *string {
	return &value
}
