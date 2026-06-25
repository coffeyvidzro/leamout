package sms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/modules/credits"
	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
	"github.com/cuffeyvidzro/leamout/internal/sms/routing"
	"github.com/google/uuid"
)

var (
	ErrProviderRequired  = errors.New("sms provider is required")
	ErrCreditsRequired   = errors.New("sms credits service is required")
	ErrRecipientRequired = errors.New("sms recipient is required")
	ErrContentRequired   = errors.New("sms content is required")
	ErrUserRequired      = errors.New("sms user is required")
)

type CreditLedger interface {
	Debit(ctx context.Context, params credits.DebitParams) (*credits.Balance, error)
	Refund(ctx context.Context, params credits.RefundParams) (*credits.Balance, error)
}

type Router interface {
	Route(rawDestination string) (routing.Route, error)
}

type Service struct {
	credits     CreditLedger
	router      Router
	providers   map[string]provider.Provider
	defaultFrom string
}

func NewService(credits CreditLedger, router Router, providers map[string]provider.Provider, cfg Config) *Service {
	return &Service{
		credits:     credits,
		router:      router,
		providers:   providers,
		defaultFrom: strings.TrimSpace(cfg.DefaultFrom),
	}
}

func (s *Service) Name() string {
	return "sms"
}

func (s *Service) Send(ctx context.Context, msg Message) error {
	if s == nil || s.credits == nil {
		return ErrCreditsRequired
	}
	normalized, err := s.normalize(msg)
	if err != nil {
		return err
	}
	if s.router == nil {
		return fmt.Errorf("sms router is required")
	}

	route, err := s.router.Route(normalized.To)
	if err != nil {
		return err
	}
	sender := s.providers[route.Provider]
	if sender == nil {
		return fmt.Errorf("%w: %s", ErrProviderRequired, route.Provider)
	}

	reference := normalized.Reference
	if reference == "" {
		reference = newReference()
	}
	metadata := defaultMetadata(normalized.Metadata)
	metadata["country_code"] = route.CountryCode
	metadata["provider"] = route.Provider

	if _, err := s.credits.Debit(ctx, credits.DebitParams{
		UserID:      normalized.UserID,
		Amount:      route.CostPesewas,
		Provider:    route.Provider,
		Destination: route.Destination,
		Reference:   reference,
		Description: "Dunning SMS",
		Metadata:    metadata,
	}); err != nil {
		return err
	}

	if err := sender.Send(ctx, provider.Message{
		To:      route.Destination,
		From:    normalized.From,
		Content: normalized.Content,
	}); err != nil {
		_, _ = s.credits.Refund(ctx, credits.RefundParams{
			UserID:      normalized.UserID,
			Amount:      route.CostPesewas,
			Provider:    route.Provider,
			Destination: route.Destination,
			Reference:   reference + ":refund",
			Description: "Dunning SMS send failure refund",
			Metadata:    metadata,
		})
		return fmt.Errorf("send sms via %s: %w", sender.Name(), err)
	}

	return nil
}

func (s *Service) normalize(msg Message) (Message, error) {
	if msg.UserID == uuid.Nil {
		return msg, ErrUserRequired
	}
	msg.To = strings.TrimSpace(msg.To)
	msg.From = strings.TrimSpace(msg.From)
	msg.Content = strings.TrimSpace(msg.Content)
	msg.Reference = strings.TrimSpace(msg.Reference)

	if msg.From == "" {
		msg.From = s.defaultFrom
	}
	if msg.To == "" {
		return msg, ErrRecipientRequired
	}
	if msg.Content == "" {
		return msg, ErrContentRequired
	}

	return msg, nil
}

func newReference() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return ""
	}

	return "sms_" + hex.EncodeToString(bytes[:])
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	clone := make(map[string]any, len(metadata)+2)
	for key, value := range metadata {
		clone[key] = value
	}

	return clone
}
