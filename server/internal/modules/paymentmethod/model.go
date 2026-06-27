package paymentmethod

type CatalogResponse struct {
	Countries []Country `json:"countries"`
}

type Country struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	CallingCode      string   `json:"calling_code"`
	DefaultCurrency  string   `json:"default_currency"`
	Currencies       []string `json:"currencies"`
	Status           string   `json:"status"`
	SupportedMethods []Method `json:"supported_methods"`
}

type Method struct {
	Type      string     `json:"type"`
	Operators []Operator `json:"operators"`
}

type Operator struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type ListParams struct {
	Country  string
	Currency string
	Method   string
	Status   string
}
