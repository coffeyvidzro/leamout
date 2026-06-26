package moolre

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/payment/provider"
)

func fromInternalInitiate(req provider.InitiatePaymentRequest, defaultAccountNumber string) (*moolreInitiatePaymentRequest, error) {
	if strings.TrimSpace(req.ExternalRef) == "" {
		return nil, fmt.Errorf("moolre payment external_ref is required")
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("moolre payment amount_minor must be greater than zero")
	}
	if !sameFold(req.Currency, "GHS") {
		return nil, provider.ErrProviderUnsupportedCurrency
	}
	if req.Country != "" && !sameFold(req.Country, "GH") {
		return nil, provider.ErrProviderUnsupportedCountry
	}
	if req.Method != "" && req.Method != provider.PaymentMethodMobileMoney {
		return nil, provider.ErrProviderUnsupportedPaymentMethod
	}
	if strings.TrimSpace(req.Customer.Phone) == "" {
		return nil, fmt.Errorf("moolre payer phone is required")
	}

	channel, err := resolveChannel(req.ProviderOptions)
	if err != nil {
		return nil, err
	}

	accountNumber := stringOption(req.ProviderOptions, "accountnumber")
	if accountNumber == "" {
		accountNumber = defaultAccountNumber
	}
	if strings.TrimSpace(accountNumber) == "" {
		return nil, fmt.Errorf("moolre accountnumber is required")
	}

	reference := strings.TrimSpace(req.Description)
	if reference == "" {
		reference = req.ExternalRef
	}

	return &moolreInitiatePaymentRequest{
		Type:          moolreRequestTypeDefault,
		Channel:       channel,
		Currency:      strings.ToUpper(strings.TrimSpace(req.Currency)),
		Payer:         normalizePhone(req.Customer.Phone),
		Amount:        formatAmount(req.AmountMinor, req.Currency),
		ExternalRef:   strings.TrimSpace(req.ExternalRef),
		OTPCode:       stringOption(req.ProviderOptions, "otpcode"),
		Reference:     reference,
		SessionID:     stringOption(req.ProviderOptions, "sessionid"),
		AccountNumber: strings.TrimSpace(accountNumber),
	}, nil
}

func fromInternalVerify(req provider.VerifyPaymentRequest, defaultAccountNumber string) (*moolreStatusRequest, error) {
	accountNumber := strings.TrimSpace(defaultAccountNumber)
	if accountNumber == "" {
		return nil, fmt.Errorf("moolre accountnumber is required")
	}

	id := strings.TrimSpace(req.ExternalRef)
	idType := "1" // 1 = unique externalref.
	if id == "" {
		id = strings.TrimSpace(req.ProviderReference)
		idType = "2" // 2 = Moolre-generated ID.
	}
	if id == "" {
		return nil, fmt.Errorf("moolre verify requires external_ref or provider_reference")
	}

	return &moolreStatusRequest{
		Type:          moolreRequestTypeDefault,
		IDType:        idType,
		ID:            id,
		AccountNumber: accountNumber,
	}, nil
}

func toInternalInitiate(req provider.InitiatePaymentRequest, resp moolreInitiatePaymentResponse, raw []byte) *provider.InitiatePaymentResponse {
	metadata := map[string]string{
		"moolre_code":   resp.Code,
		"moolre_status": resp.Status.String(),
	}
	if msg := resp.Message.String(); msg != "" {
		metadata["moolre_message"] = msg
	}
	if data := resp.DataString(); data != "" {
		metadata["moolre_data"] = data
	}

	status, nextAction := mapInitiateStatus(resp)

	return &provider.InitiatePaymentResponse{
		ProviderID:        provider.ProviderMoolre,
		ExternalRef:       req.ExternalRef,
		ProviderReference: resp.DataString(),
		Status:            status,
		NextActionType:    nextAction,
		CustomerMessage:   resp.Message.String(),
		ProviderResponse:  raw,
		Metadata:          mergeStringMaps(req.Metadata, metadata),
	}
}

func toInternalVerify(resp moolreStatusResponse, raw []byte) *provider.VerifyPaymentResponse {
	paidAt := parseMoolreTimestamp(resp.Data.Timestamp)
	status := mapTXStatus(resp.Data.TXStatus)
	amountMinor := parseAmountMinor(resp.Data.Amount, "GHS")

	metadata := map[string]string{
		"moolre_code":   resp.Code,
		"moolre_status": resp.Status.String(),
	}
	if msg := resp.Message.String(); msg != "" {
		metadata["moolre_message"] = msg
	}
	addTransactionMetadata(metadata, resp.Data)

	return &provider.VerifyPaymentResponse{
		ProviderID:        provider.ProviderMoolre,
		ExternalRef:       resp.Data.ExternalRef,
		ProviderReference: resp.Data.TransactionID,
		Status:            status,
		AmountMinor:       amountMinor,
		Currency:          "GHS",
		PaidAt:            paidAt,
		ProviderResponse:  raw,
		Metadata:          metadata,
	}
}

func mapInitiateStatus(resp moolreInitiatePaymentResponse) (provider.PaymentStatus, provider.NextActionType) {
	switch strings.ToUpper(strings.TrimSpace(resp.Code)) {
	case "TP14":
		return provider.PaymentStatusPending, provider.NextActionOTPRequired
	case "TR099":
		return provider.PaymentStatusProcessing, provider.NextActionCustomerPIN
	}

	if resp.Status.IsSuccess() {
		return provider.PaymentStatusProcessing, provider.NextActionCustomerPIN
	}
	if resp.Status.IsFailure() {
		return provider.PaymentStatusFailed, provider.NextActionNone
	}
	return provider.PaymentStatusUnknown, provider.NextActionNone
}

func mapTXStatus(status intValue) provider.PaymentStatus {
	if !status.Valid() {
		return provider.PaymentStatusUnknown
	}

	switch status.Int() {
	case 0:
		return provider.PaymentStatusPending
	case 1:
		return provider.PaymentStatusSucceeded
	case 2:
		return provider.PaymentStatusFailed
	default:
		return provider.PaymentStatusUnknown
	}
}

func resolveChannel(options map[string]any) (string, error) {
	channel := stringOption(options, "channel")
	if channel == "" {
		channel = stringOption(options, "moolre_channel")
	}
	if channel != "" {
		switch strings.TrimSpace(channel) {
		case MoolreChannelMTN, MoolreChannelTelecel, MoolreChannelAT:
			return strings.TrimSpace(channel), nil
		}
	}

	providerName := stringOption(options, "provider")
	if providerName == "" {
		providerName = stringOption(options, "network")
	}
	providerName = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(providerName), " ", "_"))

	switch providerName {
	case "MTN", "MTN_MOMO", "MTN_MOMO_GHA", "MOMO_MTN":
		return MoolreChannelMTN, nil
	case "TELECEL", "TELECEL_CASH", "VODAFONE", "VODAFONE_CASH", "VODAFONE_GHA":
		return MoolreChannelTelecel, nil
	case "AT", "AIRTELTIGO", "AIRTELTIGO_GHA", "AIRTEL_TIGO", "AIRTEL_TIGO_GHA":
		return MoolreChannelAT, nil
	default:
		return "", fmt.Errorf("moolre mobile money channel is required: use provider_options.channel 13/6/7 or provider MTN/TELECEL/AT")
	}
}

func stringOption(options map[string]any, key string) string {
	if len(options) == 0 {
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
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if math.Trunc(v) == v {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		fv := float64(v)
		if math.Trunc(fv) == fv {
			return strconv.FormatInt(int64(fv), 10)
		}
		return strconv.FormatFloat(fv, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func formatAmount(amountMinor int64, currency string) string {
	if isZeroDecimalCurrency(currency) {
		return strconv.FormatInt(amountMinor, 10)
	}
	return fmt.Sprintf("%d.%02d", amountMinor/100, amountMinor%100)
}

func parseAmountMinor(amount string, currency string) int64 {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0
	}

	parsed, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0
	}
	if isZeroDecimalCurrency(currency) {
		return int64(math.Round(parsed))
	}
	return int64(math.Round(parsed * 100))
}

func isZeroDecimalCurrency(currency string) bool {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "BIF", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF", "ZMW":
		return true
	default:
		return false
	}
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return strings.TrimPrefix(phone, "+")
}

func parseMoolreTimestamp(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	for _, layout := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		parsed, err := time.ParseInLocation(layout, value, time.UTC)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func sameFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func mergeStringMaps(base map[string]string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		if strings.TrimSpace(k) != "" {
			out[k] = v
		}
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" {
			out[k] = v
		}
	}
	return out
}

func addTransactionMetadata(metadata map[string]string, tx moolreTransaction) {
	if tx.TXStatus.String() != "" {
		metadata["txstatus"] = tx.TXStatus.String()
	}
	if tx.TXType.String() != "" {
		metadata["txtype"] = tx.TXType.String()
	}
	if tx.AccountNumber != "" {
		metadata["accountnumber"] = tx.AccountNumber
	}
	if tx.Payer != "" {
		metadata["payer"] = tx.Payer
	}
	if tx.Payee != "" {
		metadata["payee"] = tx.Payee
	}
	if tx.Amount != "" {
		metadata["amount"] = tx.Amount
	}
	if tx.Value != "" {
		metadata["value"] = tx.Value
	}
	if tx.TransactionID != "" {
		metadata["transactionid"] = tx.TransactionID
	}
	if tx.ExternalRef != "" {
		metadata["externalref"] = tx.ExternalRef
	}
	if tx.ThirdPartyRef != "" {
		metadata["thirdpartyref"] = tx.ThirdPartyRef
	}
	if tx.Timestamp != "" {
		metadata["ts"] = tx.Timestamp
	}
}
