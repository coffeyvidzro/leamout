package tola

import (
	"encoding/json"
	"fmt"
)

const (
	transactionPath       = "/transaction"
	transactionTypeCharge = "charge"
)

type TransactionRequest struct {
	Msisdn          string      `json:"msisdn"`
	Type            string      `json:"type"`
	Channel         string      `json:"channel"`
	Currency        string      `json:"currency"`
	Amount          json.Number `json:"amount"`
	SourceReference string      `json:"sourcereference"`
}

type TransactionResponse struct {
	Success   bool       `json:"success"`
	Reference string     `json:"reference,omitempty"`
	Error     *TolaError `json:"error,omitempty"`
}

type TolaError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	StatusCode int
	Body       string
	TolaError  *TolaError
}

func (e *APIError) Error() string {
	if e.TolaError != nil {
		return fmt.Sprintf("tola error %d: %s", e.TolaError.Code, e.TolaError.Message)
	}

	if e.Body != "" {
		return fmt.Sprintf("tola returned status %d: %s", e.StatusCode, e.Body)
	}

	return fmt.Sprintf("tola returned status %d", e.StatusCode)
}
