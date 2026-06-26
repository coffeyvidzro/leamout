package arkesel

import (
	"context"
	"fmt"
	"log"

	"github.com/cuffeyvidzro/leamout/internal/sms/provider"
)

type ArkeselProvider struct {
	client *Client
}

func NewProvider(client *Client) *ArkeselProvider {
	return &ArkeselProvider{client: client}
}

// Name implements the provider.Provider interface
func (p *ArkeselProvider) Name() string {
	return "arkesel"
}

func (p *ArkeselProvider) Send(ctx context.Context, msg provider.Message) error {
	arkeselReq := FromInternal(msg)

	var arkeselResp SendResponse
	err := p.client.doRequest(ctx, "POST", "/sms/send", arkeselReq, &arkeselResp)
	if err != nil {
		return fmt.Errorf("arkesel send failed: %w", err)
	}

	// You can log the result here if needed using the mapper
	result := ToInternal(&arkeselResp)
	log.Printf("Arkesel sent message: %s with status: %s", result.MessageID, result.Status)

	return nil
}
