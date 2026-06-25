package arkesel

import (
	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
)

// FromInternal maps your internal unified Message to the Arkesel API format
func FromInternal(msg provider.Message) *SendRequest {
	return &SendRequest{
		Sender:     msg.From,
		Message:    msg.Content,
		Recipients: []string{msg.To},
	}
}

// ToInternal maps the Arkesel-specific API response to the unified provider.Result
func ToInternal(resp *SendResponse) provider.Result {
	messageID := ""
	if len(resp.Data) > 0 {
		messageID = resp.Data[0].ID
	}
	return provider.Result{
		Provider:  "arkesel",
		MessageID: messageID,
		Status:    resp.Status,
	}
}
