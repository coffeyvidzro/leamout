package pawapay

import (
	"fmt"
	"time"
)

const (
	payerTypeMMO     = "MMO"
	depositAccepted  = "ACCEPTED"
	depositRejected  = "REJECTED"
	depositDuplicate = "DUPLICATE_IGNORED"
)

type DepositRequest struct {
	DepositID string   `json:"depositId"`
	Amount    string   `json:"amount"`
	Currency  string   `json:"currency"`
	Payer     PayerObj `json:"payer"`

	ClientReferenceID string              `json:"clientReferenceId,omitempty"`
	CustomerMessage   string              `json:"customerMessage,omitempty"`
	Metadata          []map[string]string `json:"metadata,omitempty"`
}

type PayerObj struct {
	Type           string     `json:"type"`
	AccountDetails AccountObj `json:"accountDetails"`
}

type AccountObj struct {
	PhoneNumber string `json:"phoneNumber"`
	Provider    string `json:"provider"`
}

type DepositResponse struct {
	DepositID     string         `json:"depositId"`
	Status        string         `json:"status"`
	Created       time.Time      `json:"created,omitempty"`
	FailureReason *FailureReason `json:"failureReason,omitempty"`
}

type FailureReason struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type APIError struct {
	StatusCode    int
	Body          string
	FailureReason *FailureReason
}

func (e *APIError) Error() string {
	if e.FailureReason != nil {
		if e.FailureReason.Code != "" && e.FailureReason.Message != "" {
			return fmt.Sprintf("pawapay error %s: %s", e.FailureReason.Code, e.FailureReason.Message)
		}

		if e.FailureReason.Message != "" {
			return fmt.Sprintf("pawapay error: %s", e.FailureReason.Message)
		}
	}

	if e.Body != "" {
		return fmt.Sprintf("pawapay returned status %d: %s", e.StatusCode, e.Body)
	}

	return fmt.Sprintf("pawapay returned status %d", e.StatusCode)
}
