package tola

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment"
)

func MapChargeRequest(payload payment.UnifiedPayload) (*TransactionRequest, error) {
	amount := strings.TrimSpace(payload.Amount)
	if amount == "" {
		return nil, fmt.Errorf("missing amount")
	}

	parsedAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tola amount %q: %w", payload.Amount, err)
	}

	if parsedAmount <= 0 {
		return nil, fmt.Errorf("invalid tola amount %q: amount must be greater than zero", payload.Amount)
	}

	return &TransactionRequest{
		Msisdn:          strings.TrimSpace(payload.PhoneNumber),
		Type:            transactionTypeCharge,
		Channel:         strings.TrimSpace(payload.Operator),
		Currency:        strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Amount:          json.Number(amount),
		SourceReference: strings.TrimSpace(payload.TransactionID),
	}, nil
}
