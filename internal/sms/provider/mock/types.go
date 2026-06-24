package mock

import "github.com/cuffeyvidzro/leamout/internal/sms/provider"

// MockRequest wraps the provider message
type MockRequest struct {
	Message provider.Message
}

type MockResponse struct {
	ProviderMsgID string
	Status        string
}
