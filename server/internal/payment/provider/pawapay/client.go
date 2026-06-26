package pawapay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
)

const (
	defaultBaseURL = "https://api.sandbox.pawapay.io"
)

type Client struct {
	BaseURL     string
	BearerToken string
	HTTPClient  *http.Client
}

type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("pawapay api error: status %d", e.StatusCode)
	}

	return fmt.Sprintf("pawapay api error: status %d: %s", e.StatusCode, string(e.Body))
}

func NewClient(cfg config.ProviderConfig) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		BaseURL:     baseURL,
		BearerToken: strings.TrimSpace(cfg.APIKey),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) InitiateDeposit(ctx context.Context, payload *PawaDepositRequest) (*PawaDepositResponse, []byte, error) {
	var result PawaDepositResponse
	raw, err := c.doRequest(ctx, http.MethodPost, "/v2/deposits", payload, &result)
	if err != nil {
		return nil, raw, err
	}

	return &result, raw, nil
}

func (c *Client) CheckDepositStatus(ctx context.Context, depositID string) (*PawaDepositStatusResponse, []byte, error) {
	path := "/v2/deposits/" + url.PathEscape(strings.TrimSpace(depositID))

	var result PawaDepositStatusResponse
	raw, err := c.doRequest(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, raw, err
	}

	return &result, raw, nil
}

func (c *Client) PredictProvider(ctx context.Context, phoneNumber string) (*PawaPredictProviderResponse, []byte, error) {
	payload := PawaPredictProviderRequest{PhoneNumber: phoneNumber}

	var result PawaPredictProviderResponse
	raw, err := c.doRequest(ctx, http.MethodPost, "/v2/predict-provider", payload, &result)
	if err != nil {
		return nil, raw, err
	}

	return &result, raw, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, payload any, result any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return raw, readErr
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return raw, &APIError{
			StatusCode: resp.StatusCode,
			Body:       raw,
		}
	}

	if result == nil || len(raw) == 0 {
		return raw, nil
	}

	if err := json.Unmarshal(raw, result); err != nil {
		return raw, err
	}

	return raw, nil
}
