package tola

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
)

const (
	defaultBaseURL = "https://apidocs.tolamobile.com"
)

type Client struct {
	BaseURL    string
	authHeader string
	HTTPClient *http.Client
}

func NewClient(cfg config.ProviderConfig) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		BaseURL:    baseURL,
		authHeader: strings.TrimSpace(cfg.APIKey),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateTransaction(ctx context.Context, payload *TransactionRequest) (*TransactionResponse, error) {
	var result TransactionResponse

	if err := c.doRequest(ctx, http.MethodPost, transactionPath, payload, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, &APIError{
			StatusCode: http.StatusOK,
			TolaError:  result.Error,
		}
	}

	return &result, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, payload any, result any) error {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	var body io.Reader

	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal tola request: %w", err)
		}

		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("create tola request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("tola mobile gateway connection timed out")
		}

		if errors.Is(ctx.Err(), context.Canceled) {
			return errors.New("tola mobile request cancelled")
		}

		return fmt.Errorf("send tola request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read tola response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(raw),
		}
	}

	if result == nil || len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, result); err != nil {
		return fmt.Errorf("decode tola response: %w", err)
	}

	return nil
}
