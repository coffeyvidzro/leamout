package geoip

import (
	"fmt"
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/config"
)

func NewGeolocationService(
	cfg *config.Config,
	log *slog.Logger,
) (*Geolocator, error) {

	var primary Provider
	var fallback Provider

	if cfg.GeoIPDatabasePath != "" {
		maxmind, err := NewMaxMindProvider(cfg.GeoIPDatabasePath)
		if err != nil {
			log.Warn(
				"failed to initialize MaxMind; continuing with fallback",
				"path", cfg.GeoIPDatabasePath,
				"error", err,
			)
		} else {
			primary = maxmind
		}
	}

	if cfg.IPInfoToken != "" {
		fallback = NewIPInfoProvider(cfg.IPInfoToken)
	}

	geolocator, err := NewGeolocator(primary, fallback, log)
	if err != nil {
		return nil, fmt.Errorf("initialize geolocation service: %w", err)
	}

	return geolocator, nil
}
