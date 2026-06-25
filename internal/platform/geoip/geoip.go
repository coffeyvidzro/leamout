package geoip

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
)

type GeoInfo struct {
	IP          string
	CountryCode string
	CountryName string
	City        string
	TimeZone    string
	Source      string
}

type Provider interface {
	Lookup(ctx context.Context, ip string) (*GeoInfo, error)
	Close() error
}

type Geolocator struct {
	primary  Provider
	fallback Provider
	log      *slog.Logger
}

func NewGeolocator(
	primary Provider,
	fallback Provider,
	log *slog.Logger,
) (*Geolocator, error) {
	if primary == nil {
		primary = fallback
		fallback = nil
	}

	if primary == nil {
		return nil, errors.New("no geolocation provider configured")
	}

	return &Geolocator{
		primary:  primary,
		fallback: fallback,
		log:      log,
	}, nil
}

func (g *Geolocator) Lookup(ctx context.Context, ipStr string) (*GeoInfo, error) {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, fmt.Errorf("invalid IP address %q: %w", ipStr, err)
	}

	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return nil, fmt.Errorf("IP address %s is not publicly geolocatable", ip)
	}

	ipStr = ip.String()

	info, err := g.primary.Lookup(ctx, ipStr)
	if err == nil {
		return info, nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	if g.fallback == nil {
		return nil, fmt.Errorf("geolocation lookup failed: %w", err)
	}

	g.log.WarnContext(
		ctx,
		"primary geolocation provider failed; trying fallback",
		"ip", ipStr,
		"error", err,
	)

	info, fallbackErr := g.fallback.Lookup(ctx, ipStr)
	if fallbackErr == nil {
		return info, nil
	}

	return nil, fmt.Errorf(
		"geolocation lookup failed: primary: %v; fallback: %w",
		err,
		fallbackErr,
	)
}

func (g *Geolocator) Close() error {
	var errs []error

	if err := g.primary.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close primary provider: %w", err))
	}

	if g.fallback != nil {
		if err := g.fallback.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close fallback provider: %w", err))
		}
	}

	return errors.Join(errs...)
}
