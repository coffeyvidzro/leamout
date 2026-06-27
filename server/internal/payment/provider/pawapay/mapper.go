package pawapay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

func MapDepositRequest(payload payment.UnifiedPayload) (*DepositRequest, error) {
	amount := strings.TrimSpace(payload.Amount)
	if amount == "" {
		return nil, fmt.Errorf("missing amount")
	}

	parsedAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid pawapay amount %q: %w", payload.Amount, err)
	}

	if parsedAmount <= 0 {
		return nil, fmt.Errorf("invalid pawapay amount %q: amount must be greater than zero", payload.Amount)
	}

	req := &DepositRequest{
		DepositID: strings.TrimSpace(payload.TransactionID),
		Amount:    amount,
		Currency:  strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Payer: PayerObj{
			Type: payerTypeMMO, // hardcoded here for pawaPay
			AccountDetails: AccountObj{
				PhoneNumber: strings.TrimSpace(payload.PhoneNumber),
				Provider:    strings.TrimSpace(payload.Operator),
			},
		},
	}

	if payload.Metadata != nil {
		req.Metadata = make([]map[string]string, 0, len(payload.Metadata))

		for key, value := range payload.Metadata {
			req.Metadata = append(req.Metadata, map[string]string{
				key: value,
			})
		}
	}

	return req, nil
}
