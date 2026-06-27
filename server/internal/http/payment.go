package http

import (
	"github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider/moolre"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider/pawapay"
	"github.com/cuffeyvidzro/leamout/internal/payment/routing"
	paywebhook "github.com/cuffeyvidzro/leamout/internal/payment/webhook"
)

func (s *Server) paymentProviders() []provider.Provider {
	return []provider.Provider{
		moolre.NewProviderFromConfig(moolre.Config{
			BaseURL:       s.cfg.Moolre.BaseURL,
			APIUser:       s.cfg.Moolre.APIUser,
			APIPubKey:     s.cfg.Moolre.APIPubKey,
			AccountNumber: s.cfg.Moolre.AccountNumber,
		}),
		pawapay.NewPawapayProvider(
			pawapay.NewClient(s.cfg.PawaPay),
		),
	}
}

func (s *Server) paymentStack(hooks payment.Hooks) (*payment.Service, *paywebhook.Handler, error) {
	providers := s.paymentProviders()

	routingService, err := routing.NewService(routing.DefaultConfig(), nil, providers...)
	if err != nil {
		return nil, nil, err
	}

	paymentService := payment.NewServiceWithDefaults(routingService, hooks)

	webhookRegistry, err := paywebhook.NewRegistry(providers...)
	if err != nil {
		return nil, nil, err
	}

	webhookHandler := paywebhook.NewHandler(webhookRegistry, paymentService, paywebhook.HandlerConfig{
		Logger:             s.log,
		ReturnEventDetails: s.cfg.IsDevelopment(),
	})

	return paymentService, webhookHandler, nil
}
