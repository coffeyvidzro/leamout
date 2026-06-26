package mock

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Client struct {
	AlwaysFail bool
	mu         sync.Mutex
	messages   []MockRequest
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

	c.mu.Lock()
	c.messages = append(c.messages, *req)
	c.mu.Unlock()

	return &MockResponse{
		ProviderMsgID: "mock_msg_12345",
		Status:        "success",
	}, nil
}

func (c *Client) Messages() []MockRequest {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make([]MockRequest, len(c.messages))
	copy(messages, c.messages)
	return messages
}
