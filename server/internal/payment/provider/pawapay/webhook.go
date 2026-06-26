package pawapay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

func ParseWebhook(_ context.Context, req provider.WebhookRequest) (*provider.WebhookEvent, error) {
	var raw PawaDepositResponse
	if err := json.Unmarshal(req.Body, &raw); err != nil {
		return nil, fmt.Errorf("parse pawapay deposit callback: %w", err)
	}

	if strings.TrimSpace(raw.DepositID) == "" {
		return nil, fmt.Errorf("%w: missing depositId in pawapay callback", provider.ErrProviderInvalidRequest)
	}

	status := mapPawaStatus(raw.Status)
	eventType := "deposit." + strings.ToLower(strings.TrimSpace(raw.Status))
	if eventType == "deposit." {
		eventType = "deposit.unknown"
	}

	providerReference := raw.ProviderTransactionID
	if providerReference == "" {
		providerReference = raw.DepositID
	}

	metadata := metadataFromDeposit(&raw)
	metadata["source"] = "pawapay"
	if hasSignatureHeaders(req) {
		metadata["signature_headers_present"] = "true"
		metadata["signature_verification"] = "not_implemented"
	}

	return &provider.WebhookEvent{
		ProviderID:        provider.ProviderPawaPay,
		EventType:         eventType,
		Status:            status,
		ExternalRef:       raw.DepositID,
		ProviderReference: providerReference,
		Verified:          false,
		Payload:           req.Body,
		Metadata:          metadata,
	}, nil
}

func hasSignatureHeaders(req provider.WebhookRequest) bool {
	return req.Headers.Get("Signature") != "" ||
		req.Headers.Get("Signature-Input") != "" ||
		req.Headers.Get("Signature-Date") != "" ||
		req.Headers.Get("Content-Digest") != ""
}
