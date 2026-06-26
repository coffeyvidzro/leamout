package routing

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrInvalidDestination     = errors.New("invalid sms destination")
	ErrUnsupportedDestination = errors.New("unsupported sms destination")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Route(rawDestination string) (Route, error) {
	destination, countryCode, err := NormalizeDestination(rawDestination)
	if err != nil {
		return Route{}, err
	}

	switch countryCode {
	case "+233":
		return Route{
			Destination: destination,
			CountryCode: countryCode,
			Provider:    ProviderArkesel,
			CostPesewas: 12,
		}, nil
	case "+234":
		return Route{
			Destination: destination,
			CountryCode: countryCode,
			Provider:    ProviderMock,
			CostPesewas: 15,
		}, nil
	default:
		return Route{}, fmt.Errorf("%w: %s", ErrUnsupportedDestination, countryCode)
	}
}

func NormalizeDestination(raw string) (destination, countryCode string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", ErrInvalidDestination
	}

	var builder strings.Builder
	for i, r := range raw {
		switch {
		case r == '+' && i == 0:
			builder.WriteRune(r)
		case unicode.IsDigit(r):
			builder.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
			continue
		default:
			return "", "", fmt.Errorf("%w: %q", ErrInvalidDestination, raw)
		}
	}

	destination = builder.String()
	if strings.HasPrefix(destination, "00") {
		destination = "+" + strings.TrimPrefix(destination, "00")
	}
	if !strings.HasPrefix(destination, "+") {
		return "", "", fmt.Errorf("%w: destination must be E.164", ErrInvalidDestination)
	}

	for _, code := range []string{"+233", "+234"} {
		if strings.HasPrefix(destination, code) {
			return destination, code, nil
		}
	}

	if len(destination) >= 4 {
		return "", "", fmt.Errorf("%w: %s", ErrUnsupportedDestination, destination[:4])
	}

	return "", "", fmt.Errorf("%w: %s", ErrUnsupportedDestination, destination)
}
