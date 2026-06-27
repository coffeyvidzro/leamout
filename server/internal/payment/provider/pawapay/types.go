package pawapay

type PawaStatus string

const (
	PawaStatusAccepted         PawaStatus = "ACCEPTED"
	PawaStatusRejected         PawaStatus = "REJECTED"
	PawaStatusDuplicateIgnored PawaStatus = "DUPLICATE_IGNORED"

	PawaStatusCompleted        PawaStatus = "COMPLETED"
	PawaStatusFailed           PawaStatus = "FAILED"
	PawaStatusProcessing       PawaStatus = "PROCESSING"
	PawaStatusEnqueued         PawaStatus = "ENQUEUED"
	PawaStatusInReconciliation PawaStatus = "IN_RECONCILIATION"

	PawaStatusFound    PawaStatus = "FOUND"
	PawaStatusNotFound PawaStatus = "NOT_FOUND"
)

type PawaNextStep string

const (
	PawaNextStepRedirectToAuthURL PawaNextStep = "REDIRECT_TO_AUTH_URL"
)

type PawaDepositRequest struct {
	DepositID            string              `json:"depositId"`
	Payer                PawaParty           `json:"payer"`
	Amount               string              `json:"amount"`
	Currency             string              `json:"currency"`
	ClientReferenceID    string              `json:"clientReferenceId,omitempty"`
	CustomerMessage      string              `json:"customerMessage,omitempty"`
	SuccessfulURL        string              `json:"successfulUrl,omitempty"`
	FailedURL            string              `json:"failedUrl,omitempty"`
	PreAuthorisationCode string              `json:"preAuthorisationCode,omitempty"`
	Metadata             []PawaMetadataField `json:"metadata,omitempty"`
}

type PawaParty struct {
	Type           string             `json:"type"`
	AccountDetails PawaAccountDetails `json:"accountDetails"`
}

type PawaAccountDetails struct {
	PhoneNumber string `json:"phoneNumber"`
	Provider    string `json:"provider,omitempty"`
}

type PawaMetadataField struct {
	FieldName  string `json:"fieldName"`
	FieldValue string `json:"fieldValue"`
	IsPII      bool   `json:"isPII"`
}

type PawaFailureReason struct {
	FailureCode    string `json:"failureCode,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`
}

type PawaDepositResponse struct {
	DepositID             string              `json:"depositId"`
	Status                string              `json:"status"`
	NextStep              string              `json:"nextStep,omitempty"`
	Amount                string              `json:"amount,omitempty"`
	Currency              string              `json:"currency,omitempty"`
	Country               string              `json:"country,omitempty"`
	Payer                 *PawaParty          `json:"payer,omitempty"`
	CustomerMessage       string              `json:"customerMessage,omitempty"`
	ClientReferenceID     string              `json:"clientReferenceId,omitempty"`
	SuccessfulURL         string              `json:"successfulUrl,omitempty"`
	FailedURL             string              `json:"failedUrl,omitempty"`
	AuthorizationURL      string              `json:"authorizationUrl,omitempty"`
	Created               string              `json:"created,omitempty"`
	ProviderTransactionID string              `json:"providerTransactionId,omitempty"`
	FailureReason         *PawaFailureReason  `json:"failureReason,omitempty"`
	Metadata              []PawaMetadataField `json:"metadata,omitempty"`
}

type PawaDepositStatusResponse struct {
	Status string               `json:"status"`
	Data   *PawaDepositResponse `json:"data,omitempty"`
}

type PawaPredictProviderRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

type PawaPredictProviderResponse struct {
	Country     string `json:"country,omitempty"`
	Provider    string `json:"provider,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}
