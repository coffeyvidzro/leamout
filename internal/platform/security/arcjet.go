package security

import (
	"time"

	"github.com/arcjet/arcjet-go"
)

func NewClient(key string) (*arcjet.Client, error) {
	return arcjet.NewClient(arcjet.Config{
		Key: key,
		Rules: []arcjet.Rule{
			arcjet.Shield(arcjet.ShieldOptions{Mode: arcjet.ModeLive}),
			arcjet.DetectBot(arcjet.BotOptions{Mode: arcjet.ModeLive, Allow: []string{}}),
			arcjet.TokenBucket(arcjet.TokenBucketOptions{
				Mode:       arcjet.ModeLive,
				Capacity:   100,
				RefillRate: 10,
				Interval:   10 * time.Second,
			}),
			arcjet.Filter(arcjet.FilterOptions{
				Mode: arcjet.ModeLive,
				Deny: []string{
					"ip.src.vpn",   // VPN services
					"ip.src.proxy", // Open proxies
					"ip.src.tor",   // Tor exit nodes
				},
			}),
		},
	})
}
