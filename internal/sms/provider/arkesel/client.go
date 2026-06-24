package arkesel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(cfg config.ProviderConfig) *Client {
	return &Client{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		HTTPClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, payload any, result any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close body", "error", err)
		}
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("arkesel api error: status code %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
