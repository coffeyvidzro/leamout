package pawapay

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

const mobileMoneyPartyType = "MMO"

func FromInternal(req provider.InitiatePaymentRequest) (*PawaDepositRequest, error) {
	if strings.TrimSpace(req.ExternalRef) == "" {
		return nil, fmt.Errorf("%w: external_ref is required", provider.ErrProviderInvalidRequest)
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("%w: amount_minor must be greater than zero", provider.ErrProviderInvalidRequest)
	}
	if strings.TrimSpace(req.Currency) == "" {
		return nil, fmt.Errorf("%w: currency is required", provider.ErrProviderInvalidRequest)
	}
	if req.Method != "" && req.Method != provider.PaymentMethodMobileMoney {
		return nil, provider.ErrProviderUnsupportedPaymentMethod
	}

	phoneNumber := normalizeMSISDN(req.Customer.Phone)
	if phoneNumber == "" {
		return nil, fmt.Errorf("%w: customer.phone is required", provider.ErrProviderInvalidRequest)
	}

	pawaProvider := stringOption(req.ProviderOptions, "provider")
	if pawaProvider == "" {
		return nil, fmt.Errorf("%w: provider_options.provider is required, for example MTN_MOMO_GHA", provider.ErrProviderInvalidRequest)
	}

	successfulURL := stringOption(req.ProviderOptions, "successful_url")
	failedURL := stringOption(req.ProviderOptions, "failed_url")

	// Let ReturnURL act as a convenient fallback for redirect-based auth flows.
	if successfulURL == "" {
		successfulURL = req.ReturnURL
	}
	if failedURL == "" {
		failedURL = req.ReturnURL
	}

	return &PawaDepositRequest{
		DepositID: req.ExternalRef,
		Payer: PawaParty{
			Type: mobileMoneyPartyType,
			AccountDetails: PawaAccountDetails{
				PhoneNumber: phoneNumber,
				Provider:    pawaProvider,
			},
		},
		Amount:               formatAmount(req.AmountMinor, req.Currency),
		Currency:             strings.ToUpper(strings.TrimSpace(req.Currency)),
		ClientReferenceID:    req.ExternalRef,
		CustomerMessage:      customerMessage(req),
		StatementDescription: statementDescription(req),
		SuccessfulURL:        successfulURL,
		FailedURL:            failedURL,
		PreAuthorisationCode: stringOption(req.ProviderOptions, "pre_authorisation_code"),
		Metadata:             metadataFrom(req.Metadata),
	}, nil
}

func ToInitiateResponse(resp *PawaDepositResponse, raw []byte) *provider.InitiatePaymentResponse {
	if resp == nil {
		return &provider.InitiatePaymentResponse{
			ProviderID:       provider.ProviderPawaPay,
			Status:           provider.PaymentStatusUnknown,
			NextActionType:   provider.NextActionNone,
			ProviderResponse: raw,
		}
	}

	metadata := metadataFromDeposit(resp)
	providerReference := resp.ProviderTransactionID
	if providerReference == "" {
		providerReference = resp.DepositID
	}

	return &provider.InitiatePaymentResponse{
		ProviderID:        provider.ProviderPawaPay,
		ExternalRef:       resp.DepositID,
		ProviderReference: providerReference,
		Status:            mapPawaStatus(resp.Status),
		NextActionType:    nextActionType(resp),
		NextActionURL:     resp.AuthorizationURL,
		CustomerMessage:   customerMessageFromDeposit(resp),
		ProviderResponse:  raw,
		Metadata:          metadata,
	}
}

func ToVerifyResponse(resp *PawaDepositStatusResponse, requestedRef string, raw []byte) *provider.VerifyPaymentResponse {
	if resp == nil {
		return &provider.VerifyPaymentResponse{
			ProviderID:        provider.ProviderPawaPay,
			ExternalRef:       requestedRef,
			ProviderReference: requestedRef,
			Status:            provider.PaymentStatusUnknown,
			ProviderResponse:  raw,
		}
	}

	if strings.EqualFold(resp.Status, string(PawaStatusNotFound)) {
		return &provider.VerifyPaymentResponse{
			ProviderID:        provider.ProviderPawaPay,
			ExternalRef:       requestedRef,
			ProviderReference: requestedRef,
			Status:            provider.PaymentStatusFailed,
			ProviderResponse:  raw,
			Metadata: map[string]string{
				"pawapay_status_check": resp.Status,
			},
		}
	}

	data := resp.Data
	if data == nil {
		return &provider.VerifyPaymentResponse{
			ProviderID:        provider.ProviderPawaPay,
			ExternalRef:       requestedRef,
			ProviderReference: requestedRef,
			Status:            provider.PaymentStatusUnknown,
			ProviderResponse:  raw,
			Metadata: map[string]string{
				"pawapay_status_check": resp.Status,
			},
		}
	}

	amountMinor, _ := parseAmountMinor(data.Amount, data.Currency)
	providerReference := data.ProviderTransactionID
	if providerReference == "" {
		providerReference = data.DepositID
	}

	var paidAt *time.Time
	if mapPawaStatus(data.Status) == provider.PaymentStatusSucceeded {
		if t, err := time.Parse(time.RFC3339, data.Created); err == nil {
			paidAt = &t
		}
	}

	return &provider.VerifyPaymentResponse{
		ProviderID:        provider.ProviderPawaPay,
		ExternalRef:       data.DepositID,
		ProviderReference: providerReference,
		Status:            mapPawaStatus(data.Status),
		AmountMinor:       amountMinor,
		Currency:          strings.ToUpper(strings.TrimSpace(data.Currency)),
		PaidAt:            paidAt,
		ProviderResponse:  raw,
		Metadata:          metadataFromDeposit(data),
	}
}

func mapPawaStatus(status string) provider.PaymentStatus {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case string(PawaStatusCompleted):
		return provider.PaymentStatusSucceeded
	case string(PawaStatusFailed), string(PawaStatusRejected), string(PawaStatusDuplicateIgnored):
		return provider.PaymentStatusFailed
	case string(PawaStatusAccepted), string(PawaStatusProcessing), string(PawaStatusEnqueued), string(PawaStatusInReconciliation):
		return provider.PaymentStatusProcessing
	case string(PawaStatusNotFound):
		return provider.PaymentStatusFailed
	default:
		return provider.PaymentStatusUnknown
	}
}

func nextActionType(resp *PawaDepositResponse) provider.NextActionType {
	if resp == nil {
		return provider.NextActionNone
	}

	if strings.EqualFold(resp.NextStep, string(PawaNextStepRedirectToAuthURL)) && strings.TrimSpace(resp.AuthorizationURL) != "" {
		return provider.NextActionRedirect
	}

	return provider.NextActionNone
}

func metadataFromDeposit(resp *PawaDepositResponse) map[string]string {
	metadata := map[string]string{}

	if resp == nil {
		return metadata
	}

	addIfNotBlank(metadata, "pawapay_status", resp.Status)
	addIfNotBlank(metadata, "next_step", resp.NextStep)
	addIfNotBlank(metadata, "authorization_url", resp.AuthorizationURL)
	addIfNotBlank(metadata, "provider_transaction_id", resp.ProviderTransactionID)
	addIfNotBlank(metadata, "country", resp.Country)
	addIfNotBlank(metadata, "customer_message", resp.CustomerMessage)
	addIfNotBlank(metadata, "client_reference_id", resp.ClientReferenceID)

	if resp.FailureReason != nil {
		addIfNotBlank(metadata, "failure_code", resp.FailureReason.FailureCode)
		addIfNotBlank(metadata, "failure_message", resp.FailureReason.FailureMessage)
	}

	if len(resp.Metadata) > 0 {
		for _, item := range resp.Metadata {
			if strings.TrimSpace(item.FieldName) == "" {
				continue
			}
			metadata["meta_"+item.FieldName] = item.FieldValue
		}
	}

	return metadata
}

func metadataFrom(metadata map[string]string) []PawaMetadataField {
	if len(metadata) == 0 {
		return nil
	}

	out := make([]PawaMetadataField, 0, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}

		out = append(out, PawaMetadataField{
			FieldName:  key,
			FieldValue: value,
			IsPII:      false,
		})
	}

	return out
}

func formatAmount(amountMinor int64, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if isZeroDecimalCurrency(currency) {
		return fmt.Sprintf("%d", amountMinor)
	}

	return fmt.Sprintf("%d.%02d", amountMinor/100, amountMinor%100)
}

func parseAmountMinor(amount string, currency string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, nil
	}

	if isZeroDecimalCurrency(currency) {
		return strconv.ParseInt(strings.Split(amount, ".")[0], 10, 64)
	}

	parts := strings.SplitN(amount, ".", 2)
	major, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	minor := int64(0)
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 2 {
			frac = frac[:2]
		}
		for len(frac) < 2 {
			frac += "0"
		}
		minor, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return major*100 + minor, nil
}

func isZeroDecimalCurrency(currency string) bool {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "BIF", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF", "ZMW":
		return true
	default:
		return false
	}
}

func normalizeMSISDN(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	return phone
}

func stringOption(options map[string]any, key string) string {
	if options == nil {
		return ""
	}

	value, ok := options[key]
	if !ok || value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func customerMessage(req provider.InitiatePaymentRequest) string {
	if v := stringOption(req.ProviderOptions, "customer_message"); v != "" {
		return v
	}
	return req.Description
}

func statementDescription(req provider.InitiatePaymentRequest) string {
	if v := stringOption(req.ProviderOptions, "statement_description"); v != "" {
		return v
	}
	return req.Description
}

func customerMessageFromDeposit(resp *PawaDepositResponse) string {
	if resp == nil {
		return ""
	}
	if message := strings.TrimSpace(resp.CustomerMessage); message != "" {
		return message
	}
	if resp.FailureReason != nil {
		if message := strings.TrimSpace(resp.FailureReason.FailureMessage); message != "" {
			return message
		}
		if code := strings.TrimSpace(resp.FailureReason.FailureCode); code != "" {
			return code
		}
	}
	if mapPawaStatus(resp.Status) == provider.PaymentStatusFailed {
		return "PawaPay rejected the payment prompt request."
	}
	return ""
}

func addIfNotBlank(metadata map[string]string, key string, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if metadata == nil || key == "" || value == "" {
		return
	}
	metadata[key] = value
}
