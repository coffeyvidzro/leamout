package geoip

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/oschwald/geoip2-golang/v2"
)

type MaxMindProvider struct {
	db *geoip2.Reader
}

func NewMaxMindProvider(dbPath string) (*MaxMindProvider, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open MaxMind database %q: %w", dbPath, err)
	}

	return &MaxMindProvider{db: db}, nil
}

func (p *MaxMindProvider) Lookup(ctx context.Context, ipStr string) (*GeoInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, fmt.Errorf("invalid IP address %q: %w", ipStr, err)
	}
	ip = ip.Unmap()

	record, err := p.db.City(ip)
	if err != nil {
		return nil, fmt.Errorf("MaxMind lookup failed: %w", err)
	}
	if !record.HasData() {
		return nil, fmt.Errorf("no MaxMind geolocation data for IP %s", ip)
	}

	return &GeoInfo{
		IP:          ip.String(),
		CountryCode: record.Country.ISOCode,
		CountryName: record.Country.Names.English,
		City:        record.City.Names.English,
		TimeZone:    record.Location.TimeZone,
		Source:      "maxmind",
	}, nil
}

func (p *MaxMindProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
