package http

import (
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	modulepayment "github.com/cuffeyvidzro/leamout/internal/modules/payment"
	"github.com/cuffeyvidzro/leamout/internal/modules/transaction"
	"github.com/cuffeyvidzro/leamout/internal/modules/wallet"
	corepayment "github.com/cuffeyvidzro/leamout/internal/payment"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider/pawapay"
	"github.com/cuffeyvidzro/leamout/internal/payment/provider/tola"
	paymentrouting "github.com/cuffeyvidzro/leamout/internal/payment/routing"
)

type paymentStack struct {
	PaymentService  *modulepayment.Service
	PaymentHandler  *modulepayment.Handler
	CheckoutService *checkout.Service
	CheckoutHandler *checkout.Handler
}

func (s *Server) paymentProviders() map[corepayment.ProviderName]corepayment.ChargeProvider {
	pawaPayProvider := pawapay.NewProvider(
		pawapay.NewClient(s.cfg.PawaPay),
	)

	tolaProvider := tola.NewProvider(
		tola.NewClient(s.cfg.Tola),
	)

	return map[corepayment.ProviderName]corepayment.ChargeProvider{
		corepayment.ProviderPawaPay: pawaPayProvider,
		corepayment.ProviderTola:    tolaProvider,
	}
}

func (s *Server) buildPaymentStack(
	checkoutRepo *checkout.Repository,
	paymentRepo *modulepayment.Repository,
	transactionService *transaction.Service,
	walletService *wallet.Service,
) *paymentStack {
	paymentRouter := paymentrouting.NewService(
		paymentrouting.NewDefaultConfig(),
		paymentrouting.NewPriorityStrategy(),
		s.paymentProviders(),
	)

	corePaymentService := corepayment.NewService(paymentRouter)

	paymentService := modulepayment.NewService(
		paymentRepo,
		corePaymentService,
		transactionService,
		walletService,
	)

	checkoutService := checkout.NewService(checkoutRepo, paymentService)
	paymentService.SetCheckoutCompleter(checkoutService)

	return &paymentStack{
		PaymentService:  paymentService,
		PaymentHandler:  modulepayment.NewHandler(paymentService),
		CheckoutService: checkoutService,
		CheckoutHandler: checkout.NewHandler(checkoutService),
	}
}
