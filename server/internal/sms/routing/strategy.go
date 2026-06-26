package routing

const (
	ProviderArkesel = "arkesel"
	ProviderMock    = "mock"
)

type Route struct {
	Destination string
	CountryCode string
	Provider    string
	CostPesewas int64
}
