package moolre

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

func ParseWebhook(ctx context.Context, req provider.WebhookRequest) (*provider.WebhookEvent, error) {
	_ = ctx

	if len(req.Body) == 0 {
		return nil, fmt.Errorf("moolre webhook body is empty")
	}

	var payload moolreWebhookPayload
	if err := json.Unmarshal(req.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode moolre webhook: %w", err)
	}

	status := mapWebhookStatus(payload)
	metadata := map[string]string{
		"moolre_code":   strings.TrimSpace(payload.Code),
		"moolre_status": payload.Status.String(),
	}
	if msg := payload.Message.String(); msg != "" {
		metadata["moolre_message"] = msg
	}
	addTransactionMetadata(metadata, payload.Data)

	return &provider.WebhookEvent{
		ProviderID:        provider.ProviderMoolre,
		EventType:         eventTypeForStatus(status),
		Status:            status,
		ExternalRef:       payload.Data.ExternalRef,
		ProviderReference: payload.Data.TransactionID,
		Verified:          false,
		Payload:           req.Body,
		Metadata:          metadata,
	}, nil
}

func mapWebhookStatus(payload moolreWebhookPayload) provider.PaymentStatus {
	if payload.Data.TXStatus.Valid() {
		return mapTXStatus(payload.Data.TXStatus)
	}

	if payload.Status.IsSuccess() {
		return provider.PaymentStatusSucceeded
	}
	if payload.Status.IsFailure() {
		return provider.PaymentStatusFailed
	}
	return provider.PaymentStatusUnknown
}

func eventTypeForStatus(status provider.PaymentStatus) string {
	switch status {
	case provider.PaymentStatusSucceeded:
		return "payment.succeeded"
	case provider.PaymentStatusPending, provider.PaymentStatusProcessing:
		return "payment.pending"
	case provider.PaymentStatusFailed:
		return "payment.failed"
	case provider.PaymentStatusCanceled:
		return "payment.canceled"
	case provider.PaymentStatusExpired:
		return "payment.expired"
	default:
		return "payment.unknown"
	}
}
