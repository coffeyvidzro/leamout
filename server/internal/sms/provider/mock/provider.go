package mock

import (
	"context"
	"fmt"

	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
)

type MockProvider struct {
	client *Client
}

func NewProvider(client *Client) *MockProvider {
	return &MockProvider{client: client}
}

func (p *MockProvider) Name() string {
	return "mock"
}

// Send satisfies the provider.Provider interface
func (p *MockProvider) Send(ctx context.Context, msg provider.Message) error {
	mockReq := FromInternal(msg)

	// Call the fake client
	_, err := p.client.DoSend(ctx, mockReq)
	if err != nil {
		return fmt.Errorf("mock provider failed: %w", err)
	}

	return nil
}
