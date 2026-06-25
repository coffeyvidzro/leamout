package routing_test

import (
	"errors"
	"testing"

	"github.com/cuffeyvidzro/leamout/internal/sms/routing"
)

func TestRouteNormalizesAndRoutesSupportedDestinations(t *testing.T) {
	service := routing.NewService()

	ghana, err := service.Route("+233 50-123-4567")
	if err != nil {
		t.Fatalf("route ghana: %v", err)
	}
	if ghana.Destination != "+233501234567" || ghana.CountryCode != "+233" || ghana.Provider != routing.ProviderArkesel || ghana.CostPesewas != 12 {
		t.Fatalf("unexpected ghana route: %+v", ghana)
	}

	nigeria, err := service.Route("00234 801 234 5678")
	if err != nil {
		t.Fatalf("route nigeria: %v", err)
	}
	if nigeria.Destination != "+2348012345678" || nigeria.CountryCode != "+234" || nigeria.Provider != routing.ProviderMock || nigeria.CostPesewas != 15 {
		t.Fatalf("unexpected nigeria route: %+v", nigeria)
	}
}

func TestRouteRejectsUnsupportedDestinations(t *testing.T) {
	service := routing.NewService()

	if _, err := service.Route("+15551234567"); !errors.Is(err, routing.ErrUnsupportedDestination) {
		t.Fatalf("expected unsupported destination, got %v", err)
	}
	if _, err := service.Route("0551234567"); !errors.Is(err, routing.ErrInvalidDestination) {
		t.Fatalf("expected invalid destination, got %v", err)
	}
}
