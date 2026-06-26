package mock

import (
	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
)

func FromInternal(msg provider.Message) *MockRequest {
	return &MockRequest{
		// Add mapping logic if MockRequest needs specific fields
		Message: msg,
	}
}

func ToInternal(mockResp *MockResponse) provider.Result {
	return provider.Result{
		Provider:  "mock",
		MessageID: mockResp.ProviderMsgID,
		Status:    mockResp.Status,
	}
}
