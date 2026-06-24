package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
)

var (
	ErrProviderRequired  = errors.New("sms provider is required")
	ErrRecipientRequired = errors.New("sms recipient is required")
	ErrContentRequired   = errors.New("sms content is required")
)

type Service struct {
	provider    provider.Provider
	defaultFrom string
}

func NewService(provider provider.Provider, cfg Config) *Service {
	return &Service{
		provider:    provider,
		defaultFrom: strings.TrimSpace(cfg.DefaultFrom),
	}
}

func (s *Service) Name() string {
	if s == nil || s.provider == nil {
		return "sms"
	}

	return s.provider.Name()
}

func (s *Service) Send(ctx context.Context, msg provider.Message) error {
	if s == nil || s.provider == nil {
		return ErrProviderRequired
	}

	normalized, err := s.normalize(msg)
	if err != nil {
		return err
	}
	if err := s.provider.Send(ctx, normalized); err != nil {
		return fmt.Errorf("send sms via %s: %w", s.provider.Name(), err)
	}

	return nil
}

func (s *Service) normalize(msg provider.Message) (provider.Message, error) {
	msg.To = strings.TrimSpace(msg.To)
	msg.From = strings.TrimSpace(msg.From)
	msg.Content = strings.TrimSpace(msg.Content)

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
