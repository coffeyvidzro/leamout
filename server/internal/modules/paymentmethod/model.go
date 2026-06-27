package paymentmethod

type CatalogResponse struct {
	Countries []Country `json:"countries"`
}

type Country struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	Prefix           string   `json:"prefix"`
	Currency         string   `json:"currency"`
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
