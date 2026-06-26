package moolre

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultBaseURL = "https://api.moolre.com"
	sandboxBaseURL = "https://sandbox.moolre.com"

	moolrePathInitiatePayment = "/open/transact/payment"
	moolrePathPaymentStatus   = "/open/transact/status"

	moolreRequestTypeDefault = 1

	MoolreChannelMTN     = "13"
	MoolreChannelTelecel = "6"
	MoolreChannelAT      = "7"
)

type Config struct {
	// BaseURL should be https://api.moolre.com for live or https://sandbox.moolre.com for sandbox.
	BaseURL string

	// APIUser is sent as X-API-USER. Moolre sandbox still requires this header.
	APIUser string

	// APIKey is the private key used for direct payment initiation in live mode.
	APIKey string

	// APIPubKey is the public key used for status verification in live mode.
	APIPubKey string

	// AccountNumber is your Moolre wallet/account number.
	AccountNumber string
}

type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(e.Body))
	}
	if e.Code != "" {
		return fmt.Sprintf("moolre api error: http=%d code=%s message=%s", e.HTTPStatus, e.Code, msg)
	}
	return fmt.Sprintf("moolre api error: http=%d message=%s", e.HTTPStatus, msg)
}

type statusValue struct {
	raw string
}

func (s *statusValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		s.raw = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.raw = strings.TrimSpace(str)
		return nil
	}

	s.raw = strings.TrimSpace(string(data))
	return nil
}

func (s statusValue) String() string {
	return s.raw
}

func (s statusValue) IsSuccess() bool {
	switch strings.ToLower(strings.TrimSpace(s.raw)) {
	case "1", "true", "success", "successful":
		return true
	default:
		return false
	}
}

func (s statusValue) IsFailure() bool {
	switch strings.ToLower(strings.TrimSpace(s.raw)) {
	case "0", "false", "failed", "failure", "error":
		return true
	default:
		return false
	}
}

type intValue struct {
	valid bool
	value int
	raw   string
}

func (i *intValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*i = intValue{}
		return nil
	}

	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		i.valid = true
		i.value = n
		i.raw = strconv.Itoa(n)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		str = strings.TrimSpace(str)
		i.raw = str
		parsed, err := strconv.Atoi(str)
		if err == nil {
			i.valid = true
			i.value = parsed
		}
		return nil
	}

	i.raw = strings.TrimSpace(string(data))
	return nil
}

func (i intValue) Int() int {
	return i.value
}

func (i intValue) Valid() bool {
	return i.valid
}

func (i intValue) String() string {
	return i.raw
}

type flexibleMessage struct {
	text string
}

func (m *flexibleMessage) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		m.text = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		m.text = strings.TrimSpace(str)
		return nil
	}

	var parts []string
	if err := json.Unmarshal(data, &parts); err == nil {
		m.text = strings.TrimSpace(strings.Join(parts, " "))
		return nil
	}

	m.text = strings.TrimSpace(string(data))
	return nil
}

func (m flexibleMessage) String() string {
	return m.text
}

type moolreInitiatePaymentRequest struct {
	Type          int    `json:"type"`
	Channel       string `json:"channel"`
	Currency      string `json:"currency"`
	Payer         string `json:"payer"`
	Amount        string `json:"amount"`
	ExternalRef   string `json:"externalref"`
	OTPCode       string `json:"otpcode,omitempty"`
	Reference     string `json:"reference,omitempty"`
	SessionID     string `json:"sessionid,omitempty"`
	AccountNumber string `json:"accountnumber"`
}

type moolreInitiatePaymentResponse struct {
	Status  statusValue     `json:"status"`
	Code    string          `json:"code"`
	Message flexibleMessage `json:"message"`
	Data    json.RawMessage `json:"data"`
	Go      json.RawMessage `json:"go"`
}

func (r moolreInitiatePaymentResponse) DataString() string {
	return rawString(r.Data)
}

type moolreStatusRequest struct {
	Type          int    `json:"type"`
	IDType        string `json:"idtype"`
	ID            string `json:"id"`
	AccountNumber string `json:"accountnumber"`
}

type moolreStatusResponse struct {
	Status  statusValue       `json:"status"`
	Code    string            `json:"code"`
	Message flexibleMessage   `json:"message"`
	Data    moolreTransaction `json:"data"`
	Go      json.RawMessage   `json:"go"`
}

type moolreTransaction struct {
	TXStatus      intValue `json:"txstatus"`
	TXType        intValue `json:"txtype"`
	AccountNumber string   `json:"accountnumber"`
	Payer         string   `json:"payer"`
	Payee         string   `json:"payee"`
	Amount        string   `json:"amount"`
	Value         string   `json:"value"`
	TransactionID string   `json:"transactionid"`
	ExternalRef   string   `json:"externalref"`
	ThirdPartyRef string   `json:"thirdpartyref"`
	Timestamp     string   `json:"ts"`
}

type moolreWebhookPayload struct {
	Status  statusValue       `json:"status"`
	Code    string            `json:"code"`
	Message flexibleMessage   `json:"message"`
	Data    moolreTransaction `json:"data"`
}

func rawString(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str)
	}

	return strings.TrimSpace(string(raw))
}
