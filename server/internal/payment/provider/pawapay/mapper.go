package pawapay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

func MapDepositRequest(payload payment.UnifiedPayload) (*DepositRequest, error) {
	amount, err := formatPawaPayAmount(payload.Amount, payload.Currency)
	if err != nil {
		return nil, err
	}

	req := &DepositRequest{
		DepositID: strings.TrimSpace(payload.TransactionID),
		Amount:    amount,
		Currency:  strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Payer: PayerObj{
			Type: payerTypeMMO,
			AccountDetails: AccountObj{
				PhoneNumber: strings.TrimSpace(payload.PhoneNumber),
				Provider:    strings.TrimSpace(payload.Operator),
			},
		},
	}

	return req, nil
}

func formatPawaPayAmount(rawAmount string, currency string) (string, error) {
	rawAmount = strings.TrimSpace(rawAmount)
	if rawAmount == "" {
		return "", fmt.Errorf("missing amount")
	}

	amountMinor, err := strconv.ParseInt(rawAmount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid pawapay amount %q: %w", rawAmount, err)
	}
	if amountMinor <= 0 {
		return "", fmt.Errorf("invalid pawapay amount %q: amount must be greater than zero", rawAmount)
	}

	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "XOF", "XAF":
		// CFA currencies do not use decimal minor units in our system.
		return strconv.FormatInt(amountMinor, 10), nil

	default:
		// Internal amount is stored in minor units.
		// Example: 5125 GHS pesewas => "51.25"
		whole := amountMinor / 100
		fraction := amountMinor % 100
		return fmt.Sprintf("%d.%02d", whole, fraction), nil
	}
}
