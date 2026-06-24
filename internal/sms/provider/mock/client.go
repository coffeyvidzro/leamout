package mock

import (
	"context"
	"fmt"
	"time"
)

type Client struct {
	AlwaysFail bool
}

func NewClient(alwaysFail bool) *Client {
	return &Client{AlwaysFail: alwaysFail}
}

// DoSend simulates an API call with latency
func (c *Client) DoSend(ctx context.Context, req *MockRequest) (*MockResponse, error) {
	// Simulate network latency
	time.Sleep(100 * time.Millisecond)

	if c.AlwaysFail {
		return nil, fmt.Errorf("mock client: simulated connection failure")
	}

	return &MockResponse{
		ProviderMsgID: "mock_msg_12345",
		Status:        "success",
	}, nil
}
