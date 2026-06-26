package moolre

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

type authMode int

const (
	authPrivateKey authMode = iota + 1
	authPublicKey
)

type Client struct {
	baseURL       string
	apiUser       string
	apiKey        string
	apiPubKey     string
	accountNumber string
	httpClient    *http.Client
}

func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		baseURL:       baseURL,
		apiUser:       strings.TrimSpace(cfg.APIUser),
		apiKey:        strings.TrimSpace(cfg.APIKey),
		apiPubKey:     strings.TrimSpace(cfg.APIPubKey),
		accountNumber: strings.TrimSpace(cfg.AccountNumber),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	if httpClient != nil {
		c.httpClient = httpClient
	}
	return c
}

func (c *Client) InitiatePayment(ctx context.Context, req provider.InitiatePaymentRequest) (*provider.InitiatePaymentResponse, error) {
	moolreReq, err := fromInternalInitiate(req, c.accountNumber)
	if err != nil {
		return nil, err
	}

	var moolreResp moolreInitiatePaymentResponse
	raw, err := c.doJSON(ctx, http.MethodPost, moolrePathInitiatePayment, moolreReq, &moolreResp, authPrivateKey)
	if err != nil {
		return nil, err
	}

	if err := moolreResponseError(moolreResp.Status, moolreResp.Code, moolreResp.Message.String(), raw); err != nil {
		return nil, err
	}

	return toInternalInitiate(req, moolreResp, raw), nil
}

func (c *Client) VerifyPayment(ctx context.Context, req provider.VerifyPaymentRequest) (*provider.VerifyPaymentResponse, error) {
	moolreReq, err := fromInternalVerify(req, c.accountNumber)
	if err != nil {
		return nil, err
	}

	var moolreResp moolreStatusResponse
	raw, err := c.doJSON(ctx, http.MethodPost, moolrePathPaymentStatus, moolreReq, &moolreResp, authPublicKey)
	if err != nil {
		return nil, err
	}

	if err := moolreResponseError(moolreResp.Status, moolreResp.Code, moolreResp.Message.String(), raw); err != nil {
		return nil, err
	}

	return toInternalVerify(moolreResp, raw), nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, result any, mode authMode) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal moolre request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create moolre request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiUser != "" {
		req.Header.Set("X-API-USER", c.apiUser)
	}

	// Moolre sandbox does not require API key/public key headers, but live does.
	// Send configured credentials when present, without forcing sandbox callers to provide them.
	switch mode {
	case authPrivateKey:
		if c.apiKey != "" {
			req.Header.Set("X-API-KEY", c.apiKey)
		}
	case authPublicKey:
		if c.apiPubKey != "" {
			req.Header.Set("X-API-PUBKEY", c.apiPubKey)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send moolre request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read moolre response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, &APIError{HTTPStatus: resp.StatusCode, Body: raw}
	}

	if result != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return raw, fmt.Errorf("decode moolre response: %w", err)
		}
	}

	return raw, nil
}

func moolreResponseError(status statusValue, code string, message string, raw []byte) error {
	code = strings.TrimSpace(code)
	upperCode := strings.ToUpper(code)

	if status.IsSuccess() {
		return nil
	}

	switch upperCode {
	case "TP13", "INP02":
		return provider.ErrProviderDuplicateReference
	case "AIN01", "AIN04":
		return provider.ErrProviderInvalidAccount
	}

	if status.IsFailure() || code != "" || strings.TrimSpace(message) != "" {
		return &APIError{HTTPStatus: http.StatusOK, Code: code, Message: message, Body: raw}
	}

	return nil
}
