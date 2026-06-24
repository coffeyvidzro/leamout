package provider

import "context"

type Provider interface {
	Send(ctx context.Context, msg Message) error
	Name() string
}
