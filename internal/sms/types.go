package sms

import (
	"context"

	"github.com/cuffeyvidzro/leamout/internal/sms/outbox"
	"github.com/google/uuid"
)

type Config struct {
	DefaultFrom string
	Outbox      OutboxStore
}

type OutboxStore interface {
	CreateOrGet(ctx context.Context, params outbox.CreateParams) (*outbox.Message, bool, error)
	MarkDebited(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkRefunded(ctx context.Context, id uuid.UUID, err error) error
}

type Message struct {
	UserID    uuid.UUID
	To        string
	Content   string
	From      string
	Reference string
	Metadata  map[string]any
}
