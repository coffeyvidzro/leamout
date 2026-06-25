package geoip

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ipinfo/go/v2/ipinfo"
)

type IPInfo struct {
	client *ipinfo.Client
}

func NewIPInfoProvider(token string) *IPInfo {
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	return &IPInfo{
		client: ipinfo.NewClient(httpClient, nil, token),
	}
}

func (p *IPInfo) Lookup(ctx context.Context, ipStr string) (*GeoInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address %q", ipStr)
	}

	// The IPinfo Go client's GetIPInfo method does not accept a context.
	// The HTTP client timeout bounds the request; the checks before and after
	// the call ensure cancellation is still returned to the caller.
	details, err := p.client.GetIPInfo(ip)
	if err != nil {
		return nil, fmt.Errorf("IPinfo lookup failed: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if details == nil {
		return nil, errors.New("IPinfo returned an empty response")
	}
	if details.Bogon {
		return nil, fmt.Errorf("IPinfo reported %s as a bogon address", ip)
	}
	if details.Country == "" && details.City == "" && details.Timezone == "" {
		return nil, fmt.Errorf("IPinfo returned no geolocation data for %s", ip)
	}

	return &GeoInfo{
		IP:          ip.String(),
		CountryCode: details.Country,
		CountryName: details.CountryName,
		City:        details.City,
		TimeZone:    details.Timezone,
		Source:      "ipinfo",
	}, nil
}

func (p *IPInfo) Close() error {
	return nil
}
